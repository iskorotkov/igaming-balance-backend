-- https://www.postgresql.org/docs/current/datatype-enum.html
create type tx_source as enum ('Game', 'Payment', 'Service');

create type tx_state as enum ('Deposit', 'Withdraw');

create table txs (
    created_at timestamptz not null default now(),
    deleted_at timestamptz default null,
    tx_id uuid not null unique,
    balance_id uuid not null,
    source tx_source not null,
    state tx_state not null,
    amount numeric not null,
    primary key (tx_id)
);

create table balances (
    balance_id uuid not null unique,
    amount numeric not null default 0,
    primary key (balance_id),
    check (amount >= 0)
);
