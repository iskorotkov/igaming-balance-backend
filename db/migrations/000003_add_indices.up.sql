CREATE INDEX idx_txs_balance_id ON txs (balance_id);

CREATE INDEX idx_txs_balance_deleted_txid ON txs (balance_id, deleted_at, tx_id DESC);

CREATE INDEX idx_txs_deleted_at ON txs (deleted_at);

CREATE INDEX idx_txs_tx_id_desc ON txs (tx_id DESC);
