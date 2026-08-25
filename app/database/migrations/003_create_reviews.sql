CREATE TABLE reviews (
  id INTEGER PRIMARY KEY,

  apprentice_id INTEGER NOT NULL,
  trainer_id INTEGER NOT NULL,

  period_start TEXT NOT NULL,
  period_end TEXT NOT NULL,

  status TEXT NOT NULL CHECK (status IN ('draft', 'submitted', 'completed')),

  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  submitted_at INTEGER,
  completed_at INTEGER,

  FOREIGN KEY(apprentice_id) REFERENCES users(id),
  FOREIGN KEY(trainer_id) REFERENCES users(id)
);
