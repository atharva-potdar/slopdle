// Typed API client for Moodle++ Go backend
const BASE_URL = '/api';

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE_URL}${endpoint}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
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
}

export interface SubmissionReportItem {
  assignmentId: number;
  assignmentName: string;
  studentName: string;
  submissionId: number;
  submissionTime?: string;
  deadline: string;
  timeliness: 'ON TIME' | 'LATE';
  score?: string;
  gradedBy: string;
}

export interface StudentGradeItem {
  firstname: string;
  assignmentid: number;
  assignment: string;
  score: string;
  feedback?: { String: string; Valid: boolean };
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
}

export interface Rubric {
  rubricid: number;
  title: string;
  department?: { String: string; Valid: boolean };
  description?: { String: string; Valid: boolean };
}

export const api = {
  // Auth
  login: (email: string, password: string) =>
    request<{ message: string; token: string; user: User }>('/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  logout: () => request('/logout', { method: 'POST' }),
  me: () => request<{ user: User; enrollments: any[]; unreadAlerts: number }>('/me'),

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
  createResource: (categoryId: number, title: string, fileUrl: string) =>
    request(`/categories/${categoryId}/resources`, {
      method: 'POST',
      body: JSON.stringify({ title, fileUrl }),
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
  submitAssignment: (assignmentId: number, contentData: string, userId?: number) =>
    request(`/assignments/${assignmentId}/submit`, {
      method: 'POST',
      body: JSON.stringify({ contentData, userId }),
    }),
  getSubmissions: (assignmentId: number) => request<any[]>(`/assignments/${assignmentId}/submissions`),
  evaluateSubmission: (submissionId: number, score: number, feedback: string, graderId?: number) =>
    request(`/submissions/${submissionId}/evaluate`, {
      method: 'POST',
      body: JSON.stringify({ score, feedback, graderId }),
    }),
  getSubmissionReport: (courseId: number) =>
    request<SubmissionReportItem[]>(`/courses/${courseId}/submissions/report`),
  getStudentGrades: (courseId: number, studentId: number) =>
    request<StudentGradeItem[]>(`/courses/${courseId}/students/${studentId}/grades`),

  // Q&A
  getQuestions: (courseId: number) => request<Question[]>(`/courses/${courseId}/questions`),
  getQuestion: (id: number) => request<{ question: Question; answers: Answer[] }>(`/questions/${id}`),
  createQuestion: (courseId: number, title: string, body: string, userId?: number) =>
    request(`/courses/${courseId}/questions`, {
      method: 'POST',
      body: JSON.stringify({ title, body, userId }),
    }),
  createAnswer: (questionId: number, body: string, userId?: number) =>
    request(`/questions/${questionId}/answers`, {
      method: 'POST',
      body: JSON.stringify({ body, userId }),
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
  createNotification: (userId: number, message: string, type = 'Announcement') =>
    request('/notifications', {
      method: 'POST',
      body: JSON.stringify({ userId, message, type }),
    }),

  // Rubrics
  getRubrics: () => request<Rubric[]>('/rubrics'),
  createRubric: (data: { title: string; department: string; description: string }) =>
    request('/rubrics', { method: 'POST', body: JSON.stringify(data) }),
  updateRubric: (id: number, data: { title: string; department: string; description: string }) =>
    request(`/rubrics/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
};
