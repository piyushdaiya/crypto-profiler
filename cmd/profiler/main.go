/*
 *
 *  *  Copyright (c) 2026 Piyush Daiya
 *  *  *
 *  *  * Permission is hereby granted, free of charge, to any person obtaining a copy
 *  *  * of this software and associated documentation files (the "Software"), to deal
 *  *  * in the Software without restriction, including without limitation the rights
 *  *  * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  *  * copies of the Software, and to permit persons to whom the Software is
 *  *  * furnished to do so, subject to the following conditions:
 *  *  *
 *  *  * The above copyright notice and this permission notice shall be included in all
 *  *  * copies or substantial portions of the Software.
 *  *  *
 *  *  * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  *  * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  *  * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  *  * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  *  * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  *  * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *  *  * SOFTWARE.
 *
 */

package main

import (
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func sanitizeLogField(s string) string {
	replacer := strings.NewReplacer(
		"\r", "_",
		"\n", "_",
		"\t", "_",
	)
	return replacer.Replace(s)
}

func validatedListenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		log.Fatal("❌ [ENGINE] Invalid PORT value")
	}

	return fmt.Sprintf(":%d", p)
}

func closeWithLog(label string, c io.Closer) {
	if err := c.Close(); err != nil {
		log.Printf("⚠️ [ENGINE] %s close failed: %v", label, err)
	}
}
func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("🔹 [ENGINE] Starting Watchlist Engine...")

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./watchlist.db"
	}

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("❌ [ENGINE] DB Error:", err)
	}
	defer closeWithLog("database", db)

	if err := db.Ping(); err != nil {
		log.Fatal("❌ [ENGINE] DB Ping Failed:", err)
	}

	initDB()

	go func() {
		log.Println("🔹 [ENGINE] Initializing Sync Loop...")
		startSyncLoop()
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/check", loggingMiddleware(checkAddressHandler))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("⚠️ [ENGINE] Health response write failed: %v", err)
		}
	})

	addr := validatedListenAddr()

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("✅ [ENGINE] Database Available & Listening on %s", sanitizeLogField(addr))

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		method := sanitizeLogField(r.Method)
		path := sanitizeLogField(r.URL.Path)

		next(w, r)

		// #nosec G706 -- request method and path are sanitized for CR/LF/TAB before logging
		log.Printf("📡 [REQ] %s %s took %v", method, path, time.Since(start))
	}
}

// --- DATABASE INIT ---
func initDB() {
	query := `
	CREATE TABLE IF NOT EXISTS sanctioned_addresses (
		address TEXT PRIMARY KEY,
		currency TEXT,
		source TEXT,
		updated_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_address ON sanctioned_addresses(address);
	CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value TEXT);
	`
	if _, err := db.Exec(query); err != nil {
		log.Fatal("❌ [ENGINE] Failed to create tables:", err)
	}
}

func checkAddressHandler(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Missing address parameter", http.StatusBadRequest)
		return
	}

	var currency, source string
	err := db.QueryRow("SELECT currency, source FROM sanctioned_addresses WHERE address = ?", address).Scan(&currency, &source)

	response := map[string]interface{}{
		"sanctioned": false,
		"address":    address,
	}

	if err == nil {
		response["sanctioned"] = true
		response["currency"] = currency
		response["source"] = source
	}

	// Simple manual JSON response
	jsonStr := fmt.Sprintf(`{"sanctioned": %v`, response["sanctioned"])
	if response["sanctioned"] == true {
		jsonStr += fmt.Sprintf(`, "currency": "%s", "source": "%s"`, currency, source)
	}
	jsonStr += `}`

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(jsonStr)); err != nil {
		log.Printf("⚠️ [ENGINE] Check response write failed: %v", err)
	}
}

// --- SYNC ENGINE ---

func startSyncLoop() {
	for {
		if shouldUpdate() {
			log.Println("⬇️  [SYNC] Update Detected. Starting OFAC Download...")
			if err := downloadAndParseOFAC(); err != nil {
				log.Printf("❌ [SYNC] Download Failed: %v", err)
			} else {
				log.Println("✅ [SYNC] Database Update Complete.")
			}
		} else {
			log.Println("✅ [SYNC] Database is up to date.")
		}
		time.Sleep(12 * time.Hour)
	}
}

func shouldUpdate() bool {
	url := "https://www.treasury.gov/ofac/downloads/sanctions/1.0/sdn_advanced.xml"

	var localLastMod string
	_ = db.QueryRow("SELECT value FROM metadata WHERE key='last_modified'").Scan(&localLastMod)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		log.Printf("⚠️ [SYNC] Could not check remote headers: %v", err)
		return true // Fail open
	}
	defer closeWithLog("head response body", resp.Body)

	remoteLastMod := resp.Header.Get("Last-Modified")
	return localLastMod != remoteLastMod
}

// --- XML STRUCTURES ---

// FeatureTypeValue represents a flattened reference value from the OFAC XML feed.
type FeatureTypeValue struct {
	ID    string `xml:"ID,attr"`
	Value string `xml:",chardata"`
}

// DistinctParty represents a sanctioned party entry in the OFAC XML feed.
type DistinctParty struct {
	Profile []Profile `xml:"Profile"`
}
type Profile struct {
	Feature []Feature `xml:"Feature"`
}
type Feature struct {
	FeatureTypeID string           `xml:"FeatureTypeID,attr"`
	Version       []FeatureVersion `xml:"FeatureVersion"`
}
type FeatureVersion struct {
	VersionDetail []VersionDetail `xml:"VersionDetail"`
}
type VersionDetail struct {
	Value string `xml:",chardata"`
}

func downloadAndParseOFAC() error {
	url := "https://www.treasury.gov/ofac/downloads/sanctions/1.0/sdn_advanced.xml"

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer closeWithLog("download response body", resp.Body)

	lastMod := resp.Header.Get("Last-Modified")
	log.Printf("🔹 [SYNC] Header Last-Modified: %s", lastMod)

	decoder := xml.NewDecoder(resp.Body)

	// PRE-FILL MAP with known IDs provided by user
	cryptoTypeMap := map[string]string{
		"344":  "XBT",
		"345":  "ETH",
		"686":  "ZEC",
		"687":  "DASH",
		"688":  "BTG",
		"689":  "ETC",
		"706":  "BSV",
		"726":  "BCH",
		"746":  "XVG",
		"992":  "TRX",
		"998":  "USDC",
		"1007": "ARB",
		"1008": "BSC",
		"1167": "SOL",
		// Additional IDs often found in OFAC data
		"573": "XMR",
		"572": "LTC",
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO sanctioned_addresses(address, currency, source, updated_at) VALUES(?, ?, 'OFAC', ?)")
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("⚠️ [SYNC] rollback failed after prepare error: %v", rbErr)
		}
		return err
	}
	defer closeWithLog("prepared statement", stmt)

	now := time.Now()
	count := 0
	loaded := 0

	log.Println("🔹 [SYNC] Parsing XML Stream...")

	for {
		t, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("⚠️ [SYNC] rollback failed after token error: %v", rbErr)
			}
			return err
		}

		switch se := t.(type) {
		case xml.StartElement:

			// STEP 1: Catch "FeatureTypeValue" (Dynamic Learning)
			// We still listen for these to catch any NEW currencies OFAC might add in the future
			if se.Name.Local == "FeatureTypeValue" {
				var ft FeatureTypeValue
				if err := decoder.DecodeElement(&ft, &se); err != nil {
					continue
				}

				if strings.Contains(ft.Value, "Digital Currency Address") {
					parts := strings.Split(ft.Value, "-")
					currency := "UNKNOWN"
					if len(parts) > 1 {
						currency = strings.TrimSpace(parts[1])
					}
					// Only add if we don't already have it hardcoded
					if _, exists := cryptoTypeMap[ft.ID]; !exists {
						cryptoTypeMap[ft.ID] = currency
						log.Printf("🔹 [SYNC] Learned new currency: ID %s = %s", ft.ID, currency)
					}
				}
			}

			// STEP 2: Scan Parties
			if se.Name.Local == "DistinctParty" {
				var p DistinctParty
				if err := decoder.DecodeElement(&p, &se); err != nil {
					continue
				}

				for _, profile := range p.Profile {
					for _, feature := range profile.Feature {
						// Is this FeatureID in our crypto map?
						if currency, isCrypto := cryptoTypeMap[feature.FeatureTypeID]; isCrypto {
							for _, v := range feature.Version {
								for _, d := range v.VersionDetail {
									addr := strings.TrimSpace(d.Value)
									if len(addr) > 10 {
										if _, err := stmt.Exec(addr, currency, now); err != nil {
											log.Printf("⚠️ [SYNC] insert failed for %s: %v", sanitizeLogField(addr), err)
										} else {
											loaded++
										}
									}
								}
							}
						}
					}
				}
				count++
				if count%10000 == 0 {
					log.Printf("🔹 [SYNC] Scanned %d Parties...", count)
				}
			}
		}
	}

	_, _ = tx.Exec("INSERT OR REPLACE INTO metadata(key, value) VALUES('last_modified', ?)", lastMod)

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("✅ [SYNC] Done. Scanned %d parties. Loaded %d sanctioned addresses.", count, loaded)

	if loaded == 0 {
		log.Println("⚠️ [SYNC] WARNING: 0 addresses loaded. Double check FeatureType IDs.")
	}

	return nil
}
