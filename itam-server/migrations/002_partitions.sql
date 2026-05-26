-- ============================================================
-- Monthly partitions for host_snapshots and config_diffs.
-- In production, use pg_partman to automate creation.
-- This file covers the initial 6-month window; extend as needed.
-- ============================================================

CREATE TABLE IF NOT EXISTS host_snapshots_2026_05 PARTITION OF host_snapshots
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE IF NOT EXISTS host_snapshots_2026_06 PARTITION OF host_snapshots
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE IF NOT EXISTS host_snapshots_2026_07 PARTITION OF host_snapshots
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE IF NOT EXISTS host_snapshots_2026_08 PARTITION OF host_snapshots
    FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE IF NOT EXISTS host_snapshots_2026_09 PARTITION OF host_snapshots
    FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE IF NOT EXISTS host_snapshots_2026_10 PARTITION OF host_snapshots
    FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');

CREATE TABLE IF NOT EXISTS config_diffs_2026_05 PARTITION OF config_diffs
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE IF NOT EXISTS config_diffs_2026_06 PARTITION OF config_diffs
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE IF NOT EXISTS config_diffs_2026_07 PARTITION OF config_diffs
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE IF NOT EXISTS config_diffs_2026_08 PARTITION OF config_diffs
    FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE IF NOT EXISTS config_diffs_2026_09 PARTITION OF config_diffs
    FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE IF NOT EXISTS config_diffs_2026_10 PARTITION OF config_diffs
    FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');

-- Default partitions catch any data that falls outside named ranges.
CREATE TABLE IF NOT EXISTS host_snapshots_default PARTITION OF host_snapshots DEFAULT;
CREATE TABLE IF NOT EXISTS config_diffs_default    PARTITION OF config_diffs    DEFAULT;
