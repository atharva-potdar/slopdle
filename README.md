# Moodle++ — Modern Database-Driven LMS

A modern Learning Management System designed with a **bottom-up, database-first architecture** built for **Div C - Batch B**:
- **Yasha Satish Patil** — 2025300174
- **Amey Pawar** — 2025300176
- **Atharva Potdar** — 2025300186

---

## Architecture Overview

```
                      +-----------------------------------+
                      |         Svelte 5 Frontend         |
                      |   (Vite + Tailwind v4 + Lucide)   |
                      |       http://localhost:5173       |
                      +-----------------+-----------------+
                                        |  REST JSON / Cookies
                                        v
                      +-----------------+-----------------+
                      |       Go 1.27 Chi Backend         |
                      |       (sqlc Generated DAO)        |
                      |       http://localhost:8080       |
                      +-----------------+-----------------+
                                        |  Unix Socket
                                        v
                      +-----------------+-----------------+
                      |         MariaDB 12.3.3            |
                      |       18 Normalized Tables        |
                      |       (4NF Relational Model)      |
                      +-----------------------------------+
```

---

## 1. Database Layer (Normalized to 4NF)

All 18 entities and weak entities from the case study:

| # | Entity / Table | Normal Form Treatment & Key Attributes |
|---|----------------|----------------------------------------|
| 1 | `User` | **PK**: `UserID`. Stores name, email, password hash, bio, GitHub & LinkedIn links. |
| 2 | `UserPhoto` | **1NF Cleared**: Multi-valued photo attribute decomposed to `(UserID, PhotoURL)`. |
| 3 | `Role` | **PK**: `RoleID`. Roles: `Teacher`, `Student`, `Teaching Assistant`. |
| 4 | `Course` | **PK**: `CourseID`, `CourseCode` (UNIQUE). Core courses. |
| 5 | `Rubric` | **PK**: `RubricID`. Standardized departmental grading rubrics. |
| 6 | `Batch` | **Weak Entity**: `(CourseID, BatchName)` UNIQUE. Sections/groups per course. |
| 7 | `ResourceCategory` | **Weak Entity**: Syllabus, Lecture Slides, Past Year Papers (PYQs). |
| 8 | `ClassSession` | **Weak Entity**: Scheduled sessions with `CHECK (EndTime > StartTime)`. |
| 9 | `Assignment` | **PK**: `AssignmentID`. Technical assignments with required format (`GITHUB`, `PY`, etc.). |
| 10 | `Resource` | **PK**: `ResourceID`. File storage URLs tied to categories. |
| 11 | `Submission` | **PK**: `SubmissionID`. Timestamped student submissions. |
| 12 | `Evaluation` | **PK**: `SubmissionID`. Score (`CHECK >= 0`), rubric feedback, grader UserID. |
| 13 | `Question` | **PK**: `QuestionID`. Q&A discussion questions. |
| 14 | `Answer` | **PK**: `AnswerID`. Upvotes and official answer status. |
| 15 | `Notification` | **PK**: `NotificationID`. In-app notification center. |
| 16 | `Prerequisite` | **M:N Recursive**: `CourseID != RequiredCourseID` with `MinPassingGrade`. |
| 17 | `Attendance` | **M:N**: `CHECK (Status IN ('Present', 'Absent', 'Excused'))`. |
| 18 | `Enrollment` | **Ternary Relation**: `User <-> Course <-> Role` with optional `BatchID`. |

---

## 2. The 3 Case Study SQL Queries

### Query 1: Bob Johnson's Grades & Evaluator Feedback
```sql
SELECT 
    u.FirstName, 
    a.Title AS Assignment, 
    e.Score, 
    e.Feedback
FROM User u
JOIN Submission s ON u.UserID = s.UserID
JOIN Assignment a ON s.AssignmentID = a.AssignmentID
JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
WHERE u.FirstName = 'Bob';
```

### Query 2: Students with Missing Submissions
```sql
SELECT 
    c.CourseCode,
    a.Title AS AssignmentName,
    a.Deadline,
    u.FirstName,
    u.LastName,
    u.Email
FROM Assignment a
JOIN Course c ON a.CourseID = c.CourseID
JOIN Enrollment e ON c.CourseID = e.CourseID
JOIN Role r ON e.RoleID = r.RoleID 
JOIN User u ON e.UserID = u.UserID
LEFT JOIN Submission s ON a.AssignmentID = s.AssignmentID AND u.UserID = s.UserID
WHERE r.RoleName = 'Student'
  AND s.SubmissionID IS NULL;
```

### Query 3: Submission Timeliness & Grader Report
```sql
SELECT 
    a.Title AS AssignmentName,
    CONCAT(Student.FirstName, ' ', Student.LastName) AS StudentName,
    s.SubmissionTime,
    a.Deadline,
    CASE 
        WHEN s.SubmissionTime > a.Deadline THEN 'LATE'
        ELSE 'ON TIME'
    END AS Timeliness,
    e.Score,
    CONCAT(Grader.FirstName, ' ', Grader.LastName) AS GradedBy
FROM Submission s
JOIN Assignment a ON s.AssignmentID = a.AssignmentID
JOIN User Student ON s.UserID = Student.UserID
JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
JOIN User Grader ON e.UserID = Grader.UserID
ORDER BY e.Score DESC;
```

---

## 3. Running the Stack

### MariaDB Migrations
```bash
mariadb -u atharva-potdar test < db/migrations/001_schema.up.sql
mariadb -u atharva-potdar test < db/migrations/002_seed.up.sql
```

### Go Backend (Port 8080)
```bash
cd backend
make build
./bin/server
```

### Svelte Frontend (Port 5173)
```bash
cd frontend
bun install
bun run dev
```

Open your browser at: **`http://localhost:5173`**
Use the top-right persona switcher to seamlessly test all roles (**Alice Smith** - Teacher, **Bob Johnson** - Student, **Charlie Brown** - Student, **Dave Miller** - TA, **Emma Watson** - Student).
