-- name: LockBalance :execrows
SELECT pg_advisory_xact_lock(hashtext((@balance_id::uuid)::text)); -- Lock a single balance row.

-- name: InsertTx :execrows
insert into txs (balance_id, source, state, amount, tx_id)
values ($1, $2, $3, $4, $5);

-- name: DeleteTxs :execrows
update txs
set deleted_at = now()
where balance_id = $1 and tx_id = any(@tx_ids::uuid[]) and deleted_at is null;

-- name: UpdateBalance :execrows
update balances
set amount = amount + $2
where balance_id = $1;

-- name: TxsByID :many
select *
from txs
where balance_id = $1 and tx_id = any(@tx_ids::uuid[]);

-- name: RecentTxs :many
select *
from txs
where balance_id = $1 and (deleted_at is null or @include_deleted::bool)
order by tx_id desc
limit $2;

-- name: PreviousTxs :many
select *
from txs
where balance_id = $1 and tx_id < $2 and (deleted_at is null or @include_deleted::bool)
order by tx_id desc
limit $3;

-- name: OpenBalance :execrows
insert into balances (balance_id, amount)
values ($1, 0);

-- name: Balance :one
select balance_id, amount
from balances
where balance_id = $1;
