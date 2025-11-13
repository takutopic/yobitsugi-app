export const myClientId = Math.random().toString(36).substring(2,10);
export const API_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080"
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