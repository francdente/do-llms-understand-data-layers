CASCADE_DELETE="""
Unsafe Delete Cascade (name: "cascade_delete")
**Definition:** A delete operation that removes rows from a child table (via explicit
application-level DELETEs or ON DELETE CASCADE) in a way that causes observable data
loss for another endpoint that needs those child rows to exist even after the parent is gone.
"""

STALE_AGGREGATE="""
Stale Aggregate (name: "stale_aggregate")
**Definition:** An endpoint modifies source-of-truth rows in a child table, but a denormalized/pre-computed aggregate value in a parent table is not recalculated. Another endpoint returns the stale aggregate, producing incorrect results.
"""

CASCADE_DELETE_FEWSHOTS="""
## Example A — issue PRESENT

Schema:
```sql
CREATE TABLE author (author_id INTEGER PRIMARY KEY, name TEXT, bio TEXT);
CREATE TABLE post (post_id INTEGER PRIMARY KEY, author_id INTEGER, title TEXT, body TEXT, published_at TEXT, FOREIGN KEY (author_id) REFERENCES author(author_id));
```

Code:
```python
@app.route("/api/authors/<int:author_id>", methods=["DELETE"])
def delete_author(author_id):
    db.execute("DELETE FROM post WHERE author_id = ?", (author_id,))
    db.execute("DELETE FROM author WHERE author_id = ?", (author_id,))
    db.commit()
    return "", 204

@app.route("/api/posts/archive")
def posts_archive():
    rows = db.execute(
        "SELECT post_id, title, published_at FROM post ORDER BY published_at"
    ).fetchall()
    return jsonify([dict(r) for r in rows])
```

Reasoning: `delete_author` removes rows from `post`. `posts_archive` reads `FROM post` with no author-existence check. After deletion, those posts vanish from the archive — observable data loss.
→ cascade_delete: detected = true

## Example B — NO issue

Schema: (same as Example A)

Code:
```python
@app.route("/api/authors/<int:author_id>", methods=["DELETE"])
def delete_author(author_id):
    db.execute("DELETE FROM post WHERE author_id = ?", (author_id,))
    db.execute("DELETE FROM author WHERE author_id = ?", (author_id,))
    db.commit()
    return "", 204

@app.route("/api/posts")
def list_posts():
    rows = db.execute(
        "SELECT p.post_id, p.title, a.name AS author_name "
        "FROM post p JOIN author a ON p.author_id = a.author_id "
        "ORDER BY p.published_at"
    ).fetchall()
    return jsonify([dict(r) for r in rows])
```

Reasoning: `delete_author` removes posts and the author. `list_posts` uses `JOIN author` — only posts whose author still exists appear. The cascade removes rows that the JOIN would have excluded anyway; the data loss is not observable through this endpoint.
→ cascade_delete: detected = false
"""

EXPOSED_RECORD="""
Exposed Record (name: "exposed_record")
**Definition:** A write endpoint changes the state of a record (or a related record) such that it should no longer be visible or available to clients, but a read endpoint still serves it without checking.
"""

EXPOSED_RECORD_FEWSHOTS="""
## Example A — issue PRESENT

Schema:
```sql
CREATE TABLE book (book_id INTEGER PRIMARY KEY, title TEXT, is_available INTEGER DEFAULT 1);
CREATE TABLE loan (loan_id INTEGER PRIMARY KEY, book_id INTEGER, borrower TEXT, due_date TEXT, FOREIGN KEY (book_id) REFERENCES book(book_id));
```

Code:
```python
@app.route("/api/books/<int:book_id>/withdraw", methods=["POST"])
def withdraw_book(book_id):
    db.execute("UPDATE book SET is_available = 0 WHERE book_id = ?", (book_id,))
    db.commit()
    return "", 204

@app.route("/api/books/catalog")
def book_catalog():
    rows = db.execute("SELECT book_id, title FROM book ORDER BY title").fetchall()
    return jsonify([dict(r) for r in rows])
```

Reasoning: `withdraw_book` sets `is_available = 0`, marking the book as withdrawn. `book_catalog` reads all books without filtering on `is_available`. Withdrawn books still appear in the catalog as if they can be borrowed.
→ exposed_record: detected = true

## Example B — NO issue

Schema: (same as Example A)

Code:
```python
@app.route("/api/books/<int:book_id>/withdraw", methods=["POST"])
def withdraw_book(book_id):
    db.execute("UPDATE book SET is_available = 0 WHERE book_id = ?", (book_id,))
    db.commit()
    return "", 204

@app.route("/api/books/catalog")
def book_catalog():
    rows = db.execute(
        "SELECT book_id, title FROM book WHERE is_available = 1 ORDER BY title"
    ).fetchall()
    return jsonify([dict(r) for r in rows])
```

Reasoning: `withdraw_book` sets `is_available = 0`. `book_catalog` filters with `WHERE is_available = 1`, so withdrawn books are properly excluded from the catalog.
→ exposed_record: detected = false
"""

STALE_AGGREGATE_FEWSHOTS="""

## Example A — issue PRESENT

Schema:
```sql
CREATE TABLE project (project_id INTEGER PRIMARY KEY, name TEXT, total_hours REAL);
CREATE TABLE task (task_id INTEGER PRIMARY KEY, project_id INTEGER, title TEXT, hours_logged REAL, FOREIGN KEY (project_id) REFERENCES project(project_id));
```

Code:
```python
@app.route("/api/tasks/<int:task_id>", methods=["PUT"])
def update_task(task_id):
    payload = request.get_json()
    db.execute(
        "UPDATE task SET hours_logged = ? WHERE task_id = ?",
        (payload["hours_logged"], task_id),
    )
    db.commit()
    return jsonify({"status": "updated"}), 200

@app.route("/api/projects/<int:project_id>")
def get_project(project_id):
    row = db.execute(
        "SELECT project_id, name, total_hours FROM project WHERE project_id = ?",
        (project_id,),
    ).fetchone()
    return jsonify(dict(row)), 200
```

Reasoning: `update_task` modifies `task.hours_logged` (source of truth) without recalculating `project.total_hours` (denormalized aggregate). `get_project` reads `total_hours` directly from the project table → returns a stale value after a task update.
→ stale_aggregate: detected = true

## Example B — NO issue

Schema: (same as Example A)

Code:
```python
@app.route("/api/tasks/<int:task_id>", methods=["PUT"])
def update_task(task_id):
    payload = request.get_json()
    db.execute(
        "UPDATE task SET hours_logged = ? WHERE task_id = ?",
        (payload["hours_logged"], task_id),
    )
    db.commit()
    return jsonify({"status": "updated"}), 200

@app.route("/api/projects/<int:project_id>")
def get_project(project_id):
    row = db.execute(
        "SELECT project_id, name FROM project WHERE project_id = ?",
        (project_id,),
    ).fetchone()
    total = db.execute(
        "SELECT COALESCE(SUM(hours_logged), 0) AS total "
        "FROM task WHERE project_id = ?",
        (project_id,),
    ).fetchone()
    result = dict(row)
    result["total_hours"] = total["total"]
    return jsonify(result), 200
```

Reasoning: `update_task` modifies `task.hours_logged` without recalculating `project.total_hours`. However, `get_project` does NOT read `total_hours` from the project table — it computes the total live via `SUM(hours_logged)` from the task table. The denormalized column's staleness is never observable through this endpoint.
→ stale_aggregate: detected = false
"""