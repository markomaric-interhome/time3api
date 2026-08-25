CREATE TABLE assignments (
  trainer_id INTEGER NOT NULL,
  apprentice_id INTEGER NOT NULL,

  created_at INTEGER NOT NULl,

  PRIMARY KEY (trainer_id, apprentice_id),
  FOREIGN KEY (trainer_id) REFERENCES users(id),
  FOREIGN KEY (apprentice_id) REFERENCES users(id)
);
