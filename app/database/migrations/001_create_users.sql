CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  team TEXT,
  role TEXT NOT NULL CHECK (role IN ('apprentice', 'trainer', 'admin')),
  email TEXT NOT NULL COLLATE NOCASE UNIQUE,
  password_hash TEXT NOT NULL,

  firstname TEXT NOT NULL,
  lastname TEXT NOT NULL,
  start_date TEXT,

  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  deleted_at INTEGER,
  last_login_at INTEGER
);
