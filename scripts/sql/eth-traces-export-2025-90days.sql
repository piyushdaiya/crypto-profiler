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

EXPORT DATA OPTIONS(
  uri='gs://YOUR_BUCKET/crypto-profiler/eth_traces_20250316_20250617_*.parquet',
  format='PARQUET',
  overwrite=true
) AS
SELECT
    block_number AS block_id,
    block_timestamp AS time,
  transaction_hash,
  transaction_index,
  trace_address AS trace_path,
  CASE
    WHEN trace_address IS NULL OR trace_address = '' THEN 0
    ELSE ARRAY_LENGTH(SPLIT(trace_address, ','))
END AS depth,
  trace_type,
  call_type,
  from_address AS sender,
  to_address AS recipient,
  value,
  input AS input_hex,
  output AS output_hex,
  gas,
  gas_used,
  subtraces AS child_call_count,
  status,
  IF(status = 0, TRUE, FALSE) AS failed,
  error AS fail_reason
FROM `bigquery-public-data.crypto_ethereum.traces`
WHERE block_timestamp >= TIMESTAMP('2025-03-16')
  AND block_timestamp < TIMESTAMP('2025-06-18');
