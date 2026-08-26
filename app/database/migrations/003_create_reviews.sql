CREATE TABLE reviews (
  id INTEGER PRIMARY KEY,

  trainer_id INTEGER NOT NULL,
  apprentice_id INTEGER NOT NULL,

  semester INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft', 'submitted', 'completed')),

  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  submitted_at INTEGER,
  completed_at INTEGER,

  FOREIGN KEY(trainer_id) REFERENCES users(id),
  FOREIGN KEY(apprentice_id) REFERENCES users(id)
);
