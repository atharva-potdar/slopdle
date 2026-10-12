// Typed API client for Moodle++ Go backend
const BASE_URL = '/api';

// sqlc NullString serializes as { String, Valid }. Helpers unwrap it.
export interface NullString {
  String: string;
  Valid: boolean;
}

export function nullStr(v: NullString | string | null | undefined): string | null {
  if (!v) return null;
  if (typeof v === 'string') return v || null;
  return v.Valid && v.String ? v.String : null;
}

export function nullStrIds(v: NullString | string | null | undefined): number[] {
  const s = nullStr(v);
  if (!s) return [];
  return s
    .split(',')
    .map((x) => parseInt(x.trim()))
    .filter((x) => !isNaN(x));
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE_URL}${endpoint}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      // ngrok free tier serves an interstitial warning page to browsers;
      // this header bypasses it for API calls. Dev tunnels only.
      ...(import.meta.env.DEV ? { 'ngrok-skip-browser-warning': 'true' } : {}),
      ...options.headers,
    },
  });

  const data = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error(data?.error || `Request failed with status ${res.status}`);
  }
  return data as T;
}

export interface User {
  userId: number;
  firstName: string;
  lastName: string;
  email: string;
  bio?: string;
  githubLink?: string;
  linkedinLink?: string;
  photos?: string[];
}

export interface Course {
  courseid: number;
  coursecode: string;
  title: string;
  description?: { String: string; Valid: boolean };
}

export interface Assignment {
  assignmentid: number;
  title: string;
  description?: { String: string; Valid: boolean };
  deadline: string;
  formatrequired?: { String: string; Valid: boolean };
  courseid: number;
  rubricid?: { Int32: number; Valid: boolean };
  rubrictitle?: { String: string; Valid: boolean };
  batchids?: NullString | null;
  batchnames?: NullString | null;
  latepenaltyperday?: string;
  latepenaltyenabled?: boolean;
  maxuploadmb?: number;
  maxuploadfiles?: number;
}

export interface SubmissionReportItem {
  assignmentId: number;
  assignmentName: string;
  studentId: number;
  studentName: string;
  submissionId: number;
  fileName?: string | null;
  submissionTime?: string;
  content: string;
  deadline: string;
  timeliness: 'ON TIME' | 'LATE';
  score?: string;
  daysLate: number;
  penaltyPerDay: string;
  penaltyEnabled: boolean;
  penaltyApplied?: string;
  effectiveScore?: string;
  gradedBy: string;
}

export interface StudentGradeItem {
  firstname: string;
  assignmentid: number;
  assignment: string;
  score: string;
  feedback?: { String: string; Valid: boolean };
  dayslate?: number;
  penaltyapplied?: string;
  effectivescore?: string;
  criterionbreakdown?: { String: string; Valid: boolean };
}

export interface PendingStudent {
  coursecode: string;
  assignmentname: string;
  deadline: string;
  userid: number;
  firstname: string;
  lastname: string;
  email: string;
}

export interface Question {
  questionid: number;
  title: string;
  body: string;
  createdat?: { Time: string; Valid: boolean };
  courseid: number;
  userid: number;
  firstname: string;
  lastname: string;
  answercount: number;
  hasofficialanswer: number;
}

export interface Answer {
  answerid: number;
  body: string;
  upvotes?: { Int32: number; Valid: boolean };
  isofficialanswer?: { Bool: boolean; Valid: boolean };
  createdat?: { Time: string; Valid: boolean };
  questionid: number;
  userid: number;
  firstname: string;
  lastname: string;
  email: string;
  userrole?: { String: string; Valid: boolean };
  hasupvoted: boolean;
}

export interface ResourceCategory {
  categoryid: number;
  categoryname: string;
  courseid: number;
}

export interface Resource {
  resourceid: number;
  title: string;
  fileurl: string;
  uploadedat?: { Time: string; Valid: boolean };
  categoryid: number;
  categoryname?: string;
  batchids?: NullString | null;
  batchnames?: NullString | null;
}

export interface StudentSubmission {
  submissionid: number;
  submissiontime?: { Time: string; Valid: boolean };
  contentdata: string;
  assignmentid: number;
  userid: number;
  score?: { String: string; Valid: boolean };
  feedback?: { String: string; Valid: boolean };
}

export interface UserSubmission {
  coursecode: string;
  coursetitle: string;
  assignmentid: number;
  assignmentname: string;
  deadline: string;
  submissionid: number;
  submissiontime?: { Time: string; Valid: boolean };
  score?: { String: string; Valid: boolean };
  effectivescore?: string;
}

export interface AttendanceSummary {
  percentage: string;
  percentageRaw: number;
  totalSessions: number;
  presentCount: number;
  absentCount: number;
  excusedCount: number;
  sessions: {
    sessionId: number;
    startTime: string;
    endTime: string;
    roomLocation: string;
    batchName: string;
    attendanceStatus: string;
  }[];
}

export interface NotificationItem {
  notificationid: number;
  message: string;
  type?: { String: string; Valid: boolean };
  isread?: { Bool: boolean; Valid: boolean };
  createdat?: { Time: string; Valid: boolean };
  userid: number;
  intenttype?: { String: string; Valid: boolean };
  intentid?: { Int32: number; Valid: boolean };
}

export interface ParsedIntent {
  tab: 'dashboard' | 'courses' | 'assignments' | 'qa';
  courseId?: number;
  assignmentId?: number;
  questionId?: number;
}

// Intent targets: announcement -> course, assignment -> assignment,
// grade -> submission, answer -> question. Course is resolved lazily
// by the caller (one fetch) except announcements which carry it.
export function parseIntent(n: NotificationItem): { kind: string; id: number } | null {
  const t = n.intenttype?.Valid ? n.intenttype.String : '';
  const id = n.intentid?.Valid ? n.intentid.Int32 : null;
  if (!t || id == null) return null;
  if (!['announcement', 'assignment', 'grade', 'answer'].includes(t)) return null;
  return { kind: t, id };
}

export interface Rubric {
  rubricid: number;
  title: string;
  department?: { String: string; Valid: boolean };
  description?: { String: string; Valid: boolean };
}

export interface Criterion {
  criterionid: number;
  rubricid: number;
  title: string;
  maxpoints: string;
}

export interface CriterionScore {
  submissionid: number;
  criterionid: number;
  score: string;
  feedback?: { String: string; Valid: boolean };
  criteriontitle: string;
  maxpoints: string;
}

export interface CriterionInput {
  criterionId?: number;
  title: string;
  maxPoints: number;
}

export interface CriterionGradeInput {
  criterionId: number;
  score: number;
  feedback?: string;
}

export const api = {
  // Auth
  login: (email: string, password: string) =>
    request<{ message: string; token: string; user: User }>('/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  logout: () => request('/logout', { method: 'POST' }),
  me: () => request<{ user: User | null; enrollments: any[]; unreadAlerts: number }>('/me'),

  // Users
  getUsers: () => request<User[]>('/users'),
  getUser: (id: number) => request<User>(`/users/${id}`),
  updateProfile: (id: number, data: Partial<User>) =>
    request(`/users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  addUserPhoto: (id: number, photoUrl: string) =>
    request(`/users/${id}/photos`, { method: 'POST', body: JSON.stringify({ photoUrl }) }),
  deleteUserPhoto: (id: number, photoUrl: string) =>
    request(`/users/${id}/photos`, { method: 'DELETE', body: JSON.stringify({ photoUrl }) }),

  // Courses
  getCourses: () => request<Course[]>('/courses'),
  getCatalog: () => request<Course[]>('/catalog'),
  getCourse: (id: number) => request<{ course: Course; prerequisites: any[] }>(`/courses/${id}`),
  createCourse: (data: { courseCode: string; title: string; description: string }) =>
    request('/courses', { method: 'POST', body: JSON.stringify(data) }),
  getPrerequisites: (courseId: number) => request<any[]>(`/courses/${courseId}/prerequisites`),
  addPrerequisite: (courseId: number, requiredCourseId: number, minPassingGrade: number) =>
    request(`/courses/${courseId}/prerequisites`, {
      method: 'POST',
      body: JSON.stringify({ requiredCourseId, minPassingGrade }),
    }),
  deletePrerequisite: (courseId: number, requiredCourseId: number) =>
    request(`/courses/${courseId}/prerequisites/${requiredCourseId}`, { method: 'DELETE' }),

  // Enrollments
  getEnrollments: (courseId: number) => request<any[]>(`/courses/${courseId}/enrollments`),
  enroll: (courseId: number, userId: number, roleId: number, batchId?: number) =>
    request(`/courses/${courseId}/enroll`, {
      method: 'POST',
      body: JSON.stringify({ userId, roleId, batchId }),
    }),
  unenroll: (courseId: number, userId: number) =>
    request(`/courses/${courseId}/enroll/${userId}`, { method: 'DELETE' }),
  updateEnrollmentBatch: (enrollmentId: number, batchId: number | null) =>
    request(`/enrollments/${enrollmentId}/batch`, {
      method: 'PUT',
      body: JSON.stringify({ batchId }),
    }),

  // Batches
  getBatches: (courseId: number) => request<any[]>(`/courses/${courseId}/batches`),
  createBatch: (courseId: number, batchName: string) =>
    request(`/courses/${courseId}/batches`, {
      method: 'POST',
      body: JSON.stringify({ batchName }),
    }),
  deleteBatch: (batchId: number) => request(`/batches/${batchId}`, { method: 'DELETE' }),

  // Resources
  getCategories: (courseId: number) => request<ResourceCategory[]>(`/courses/${courseId}/categories`),
  createCategory: (courseId: number, categoryName: string) =>
    request(`/courses/${courseId}/categories`, {
      method: 'POST',
      body: JSON.stringify({ categoryName }),
    }),
  getResources: (courseId: number) => request<Resource[]>(`/courses/${courseId}/resources`),
  createResource: (categoryId: number, title: string, fileUrl: string, batchIds: number[] = []) =>
    request(`/categories/${categoryId}/resources`, {
      method: 'POST',
      body: JSON.stringify({ title, fileUrl, batchIds }),
    }),
  deleteResource: (resourceId: number) => request(`/resources/${resourceId}`, { method: 'DELETE' }),

  // Assignments
  getAssignments: (courseId: number) => request<Assignment[]>(`/courses/${courseId}/assignments`),
  getAssignment: (id: number) => request<Assignment>(`/assignments/${id}`),
  createAssignment: (courseId: number, data: any) =>
    request(`/courses/${courseId}/assignments`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateAssignment: (id: number, data: any) =>
    request(`/assignments/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteAssignment: (id: number) => request(`/assignments/${id}`, { method: 'DELETE' }),
  getPendingStudents: (assignmentId: number) =>
    request<PendingStudent[]>(`/assignments/${assignmentId}/pending`),
  getPendingStudentsForCourse: (courseId: number) =>
    request<PendingStudent[]>(`/courses/${courseId}/pending-students`),
  getPendingAssignmentsForStudent: (userId: number) =>
    request<any[]>(`/students/${userId}/pending-assignments`),

  // Submissions & Grading
  submitAssignment: (assignmentId: number, contentData: string) =>
    request(`/assignments/${assignmentId}/submit`, {
      method: 'POST',
      body: JSON.stringify({ contentData }),
    }),
  // File upload needs multipart + progress: XHR, not fetch.
  uploadSubmission: (assignmentId: number, file: File, onProgress?: (pct: number) => void) =>
    new Promise<{ message: string; submissionIds: number[] }>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open('POST', `${BASE_URL}/assignments/${assignmentId}/upload`);
      xhr.withCredentials = true;
      // Same ngrok free-tier bypass as request() above (dev tunnels only).
      if (import.meta.env.DEV) {
        xhr.setRequestHeader('ngrok-skip-browser-warning', 'true');
      }
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100));
      };
      xhr.onload = () => {
        try {
          const data = JSON.parse(xhr.responseText);
          if (xhr.status >= 200 && xhr.status < 300) resolve(data);
          else reject(new Error(data?.error || `Upload failed (${xhr.status})`));
        } catch (e) {
          reject(new Error(`Upload failed (${xhr.status})`));
        }
      };
      xhr.onerror = () => reject(new Error('Upload failed (network)'));
      const fd = new FormData();
      fd.append('files', file, file.name);
      xhr.send(fd);
    }),
  submissionFileUrl: (submissionId: number) => `${BASE_URL}/submissions/${submissionId}/file`,
  getSubmissions: (assignmentId: number) => request<any[]>(`/assignments/${assignmentId}/submissions`),
  getSubmission: (id: number) => request<any>(`/submissions/${id}`),
  evaluateSubmission: (submissionId: number, score: number, feedback: string, graderId?: number, criteria?: CriterionGradeInput[]) =>
    request(`/submissions/${submissionId}/evaluate`, {
      method: 'POST',
      body: JSON.stringify({ score, feedback, graderId, criteria: criteria || [] }),
    }),
  getSubmissionReport: (courseId: number) =>
    request<SubmissionReportItem[]>(`/courses/${courseId}/submissions/report`),
  getStudentGrades: (courseId: number, studentId: number) =>
    request<StudentGradeItem[]>(`/courses/${courseId}/students/${studentId}/grades`),
  getStudentSubmissions: (courseId: number, studentId: number) =>
    request<StudentSubmission[]>(`/courses/${courseId}/students/${studentId}/submissions`),
  getUserSubmissions: (userId: number) =>
    request<UserSubmission[]>(`/students/${userId}/submissions`),

  // Q&A
  getQuestions: (courseId: number) => request<Question[]>(`/courses/${courseId}/questions`),
  getQuestion: (id: number) => request<{ question: Question; answers: Answer[] }>(`/questions/${id}`),
  createQuestion: (courseId: number, title: string, body: string) =>
    request(`/courses/${courseId}/questions`, {
      method: 'POST',
      body: JSON.stringify({ title, body }),
    }),
  createAnswer: (questionId: number, body: string) =>
    request(`/questions/${questionId}/answers`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    }),
  upvoteAnswer: (answerId: number) =>
    request(`/answers/${answerId}/upvote`, { method: 'PUT' }),
  markOfficialAnswer: (answerId: number, isOfficial: boolean) =>
    request(`/answers/${answerId}/official`, {
      method: 'PUT',
      body: JSON.stringify({ isOfficial }),
    }),

  // Attendance
  getCourseSessions: (courseId: number) => request<any[]>(`/courses/${courseId}/sessions`),
  createSession: (batchId: number, startTime: string, endTime: string, roomLocation: string) =>
    request(`/batches/${batchId}/sessions`, {
      method: 'POST',
      body: JSON.stringify({ startTime, endTime, roomLocation }),
    }),
  recordAttendance: (sessionId: number, userId: number, status: string) =>
    request(`/sessions/${sessionId}/attendance`, {
      method: 'POST',
      body: JSON.stringify({ userId, status }),
    }),
  recordBulkAttendance: (sessionId: number, records: { userId: number; status: string }[]) =>
    request(`/sessions/${sessionId}/attendance/bulk`, {
      method: 'POST',
      body: JSON.stringify({ records }),
    }),
  getStudentAttendance: (courseId: number, studentId: number) =>
    request<AttendanceSummary>(`/courses/${courseId}/students/${studentId}/attendance`),

  // Notifications
  getNotifications: (userId: number) =>
    request<NotificationItem[]>(`/notifications?userId=${userId}`),
  markNotificationRead: (id: number, userId?: number) =>
    request(`/notifications/${id}/read${userId ? `?userId=${userId}` : ''}`, { method: 'PUT' }),
  markAllNotificationsRead: (userId: number) =>
    request(`/notifications/read-all?userId=${userId}`, { method: 'PUT' }),
  createNotification: (userId: number, message: string, type = 'Announcement', intentType?: string, intentId?: number) =>
    request('/notifications', {
      method: 'POST',
      body: JSON.stringify({ userId, message, type, intentType, intentId }),
    }),

  // Rubrics
  getRubrics: () => request<Rubric[]>('/rubrics'),
  createRubric: (data: { title: string; department: string; description: string }) =>
    request('/rubrics', { method: 'POST', body: JSON.stringify(data) }),
  updateRubric: (id: number, data: { title: string; department: string; description: string }) =>
    request(`/rubrics/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  getRubricCriteria: (id: number) => request<Criterion[]>(`/rubrics/${id}/criteria`),
  setRubricCriteria: (id: number, criteria: CriterionInput[]) =>
    request(`/rubrics/${id}/criteria`, { method: 'PUT', body: JSON.stringify({ criteria }) }),
  getSubmissionCriteria: (submissionId: number) =>
    request<CriterionScore[]>(`/submissions/${submissionId}/criteria`),
};
