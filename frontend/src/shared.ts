const CLIENT_ID_STORAGE_KEY = 'yobitsugi-client-id';

const generateClientId = () => Math.random().toString(36).substring(2, 10);

const resolveClientId = () => {
  if (typeof window === 'undefined' || !window.localStorage) {
    return generateClientId();
  }

  const stored = window.localStorage.getItem(CLIENT_ID_STORAGE_KEY);
  if (stored) {
    return stored;
  }

  const newId = generateClientId();
  window.localStorage.setItem(CLIENT_ID_STORAGE_KEY, newId);
  return newId;
};

export const myClientId = resolveClientId();
export const API_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";
// Interface for assessment results
export interface AssessmentResult {
  id: number
  status: string;
  assessedTruth: boolean;
  confidence: number;
  reasoning: string;
}

export interface AssessmentJob {
  // GORM model fields
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;

  // Custom fields
  ClientID: string;
  Status: string; // "PENDING", "COMPLETE", "ERROR"

  // Request data
  OriginalCode: string;
  PatchedCode: string;
  BugDescription: string;

  // Result data
  ResultStatus: string;
  ResultAssessedTruth: boolean;
  ResultConfidence: number;
  ResultReasoning: string;
}