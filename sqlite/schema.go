package sqlite

var schema = []string{
	// Create a table. And an external content fts5 table to index it.
	`CREATE TABLE IF NOT EXISTS tuples(id INTEGER PRIMARY KEY, tuple TEXT, ord INTEGER, author TEXT, UNIQUE(tuple, author));`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS tuples_idx USING fts5(tuple, author, content='tuples', content_rowid='rowid');`,
	// Triggers to keep the FTS index up to date.
	`CREATE TRIGGER IF NOT EXISTS tuples_ai AFTER INSERT ON tuples BEGIN
	INSERT INTO tuples_idx(rowid, tuple, author) VALUES (new.rowid, new.tuple, new.author);
END;`,

	`CREATE TRIGGER IF NOT EXISTS tuples_ad AFTER DELETE ON tuples BEGIN
	INSERT INTO tuples_idx(fts_idx, rowid, tuple, author) VALUES('delete', old.rowid, old.tuple, old.author);
END;`,

	`CREATE TRIGGER IF NOT EXISTS tuples_au AFTER UPDATE ON tuples BEGIN
	INSERT INTO tuples_idx(tuples_idx, rowid, tuple, author) VALUES('delete', old.rowid, old.tuple, old.author);
	INSERT INTO tuples_idx(rowid, tuple, author) VALUES (new.rowid, new.tuple, new.author);
END;`,

	`CREATE TABLE IF NOT EXISTS suffixes(id INTEGER PRIMARY KEY, tuple INTEGER, word TEXT, count INTEGER, UNIQUE(tuple, word));`,
}
