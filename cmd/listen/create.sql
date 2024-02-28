CREATE TABLE IF NOT EXISTS actor(
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	name TEXT,
	username TEXT,
	published INTEGER,
	summary TEXT
);

CREATE TABLE IF NOT EXISTS activity(
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	name TEXT,
	published INTEGER,
	summary TEXT,
	content TEXT,
	attributedTo REFERENCES actor(id),
	inReplyTo INTEGER,
	object INTEGER
);

CREATE TABLE IF NOT EXISTS recipient_to(
	activity_id REFERENCES activity(id),
	rcpt REFERENCES actor(id)
);

CREATE TABLE IF NOT EXISTS recipient_cc (
	activity_id REFERENCES activity(id),
	rcpt REFERENCES actor(id)
);

CREATE VIRTUAL TABLE post USING fts5(
	id, -- AcitivityPub ID
	from,
	to,
	date,
	in_reply_to,
	body
);
