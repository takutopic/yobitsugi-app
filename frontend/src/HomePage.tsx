import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import type { AssessmentJob, AssessmentResult } from './shared'
import { myClientId, API_URL } from './shared'

const FORM_STORAGE_KEY = 'yobitsugi-form-state';
const JOBS_STORAGE_KEY = 'yobitsugi-job-history';

type StoredFormState = {
  bugDescription: string;
  originalCode: string;
  patchedCode: string;
};

const defaultFormState: StoredFormState = {
  bugDescription: '',
  originalCode: '',
  patchedCode: '',
};

let cachedFormState: StoredFormState | null = null;
let cachedJobsState: AssessmentJob[] | null = null;

const isBrowserEnv = () => typeof window !== 'undefined' && !!window.localStorage;

const loadFormFromStorage = (): StoredFormState => {
  if (cachedFormState) {
    return cachedFormState;
  }

  if (!isBrowserEnv()) {
    cachedFormState = { ...defaultFormState };
    return cachedFormState;
  }

  try {
    const raw = window.localStorage.getItem(FORM_STORAGE_KEY);
    if (!raw) {
      cachedFormState = { ...defaultFormState };
    } else {
      const parsed = JSON.parse(raw) as Partial<StoredFormState>;
      cachedFormState = { ...defaultFormState, ...parsed };
    }
  } catch (error) {
    console.warn('Failed to read stored form state:', error);
    cachedFormState = { ...defaultFormState };
  }

  return cachedFormState;
};

const loadJobsFromStorage = (): AssessmentJob[] => {
  if (cachedJobsState) {
    return [...cachedJobsState];
  }

  if (!isBrowserEnv()) {
    cachedJobsState = [];
    return [];
  }

  try {
    const raw = window.localStorage.getItem(JOBS_STORAGE_KEY);
    if (!raw) {
      cachedJobsState = [];
    } else {
      const parsed = JSON.parse(raw);
      cachedJobsState = Array.isArray(parsed) ? parsed : [];
    }
  } catch (error) {
    console.warn('Failed to read stored jobs:', error);
    cachedJobsState = [];
  }

  return [...cachedJobsState];
};

const persistFormToStorage = (state: StoredFormState) => {
  if (!isBrowserEnv()) {
    return;
  }
  try {
    window.localStorage.setItem(FORM_STORAGE_KEY, JSON.stringify(state));
  } catch (error) {
    console.warn('Failed to persist form state:', error);
  }
};

const persistJobsToStorage = (jobs: AssessmentJob[]) => {
  if (!isBrowserEnv()) {
    return;
  }
  try {
    window.localStorage.setItem(JOBS_STORAGE_KEY, JSON.stringify(jobs));
  } catch (error) {
    console.warn('Failed to persist jobs:', error);
  }
};

export function HomePage() {
  const storedForm = loadFormFromStorage();
  const [message, setMessage] = useState('Loading message from Go...');
  const [jobs, setJobs] = useState<AssessmentJob[]>(() => loadJobsFromStorage());
  const [bugDescription, setBugDescription] = useState(storedForm.bugDescription);
  const [originalCode, setOriginalCode] = useState(storedForm.originalCode);
  const [patchedCode, setPatchedCode] = useState(storedForm.patchedCode);
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    persistFormToStorage({ bugDescription, originalCode, patchedCode });
  }, [bugDescription, originalCode, patchedCode]);

  useEffect(() => {
    persistJobsToStorage(jobs);
  }, [jobs]);

  // useEffect handles the initial "hello" fetch.
  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(`${API_URL}/api/hello`);

        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json()

        setMessage(data.text);

      } catch (error) {
        console.error("Failed to fetch data:", error)
        setMessage("Failed to fetch message from Go BFF. Is it running?");
      }
    };
    
    fetchData();
  }, []);

  useEffect(() => {
    const fetchJobHistory = async() => {
      try {
        const response = await fetch(`${API_URL}/api/jobs?clientId=${myClientId}`);
        if (!response.ok) {
          throw new Error('Failed to fetch job history');
        }
        const data: AssessmentJob[] = await response.json();
        setJobs(data);

      } catch(error) {
        console.error("Failed to fetch jobs:", error);
      }
    };

    fetchJobHistory();
  }, []);

  // useEffect handles the WebSocket connection.
  useEffect(() => {
    // Create a WebSocket connection for a client
    const wsHost = new URL(API_URL).host;
    const wsUrl = `ws://${wsHost}/ws?clientId=${myClientId}`;
    console.log(`Connecting to WebSocket as: ${myClientId}`);
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {console.log('WebSocket connection established.');};

    ws.onclose = () => {console.log('WebSocket connection closed.');};

    ws.onerror = (error) => {console.error('WebSocket error:', error);};

    ws.onmessage = (event) => {
      console.log('WebSocket message received:', event.data);
      setIsLoading(false);

      try {
        const result: AssessmentResult = JSON.parse(event.data);

        setJobs(prevJobs =>
          prevJobs.map(job => {
            if (job.ID === result.id) {
              return {
                ...job,
                Status: result.status === "ERROR" ? "ERROR" : "COMPLETE",
                ResultStatus: result.status,
                ResultAssessedTruth: result.assessedTruth,
                ResultConfidence: result.confidence,
                ResultReasoning: result.reasoning,
                UpdatedAt: new Date().toISOString(),
              };
            }
            return job;
          })
        );
        
      } catch (error) {
        console.error('Failed to parse WebSocket JSON:', error);
        // setAssessment(`Error: Failed to parse message from server: ${event.data}`);
      }
    };
    ws.onclose = () => console.log('WebSocket connection closed.');
    ws.onerror = (error) => console.error('WebSocket error:', error);

    // Clean up the connection when the component unmounts
    return () => {
      ws.close();
    };
  }, []);
  
  // Submit Handler
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    // setAssessment("Assessment in progress...");

    const body = {
      bugDescription,
      originalCode,
      patchedCode,
      clientId: myClientId,
    };

    try {
      const response = await fetch(`${API_URL}/api/assess-kantei`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      console.log('Job accepted by server:', data.status);

      // API currently returns the identifier as `jobID`; fall back to `jobId` for safety
      const jobIdFromApi = data.jobID ?? data.jobId;
      const jobId = Number(jobIdFromApi);
      if (!jobIdFromApi || Number.isNaN(jobId)) {
        throw new Error('Server response missing job ID');
      }

      const newJob: AssessmentJob = {
        ID: jobId,
        CreatedAt: new Date().toISOString(),
        UpdatedAt: new Date().toISOString(),
        ClientID: myClientId,
        Status: "PENDING",
        OriginalCode: originalCode,
        PatchedCode: patchedCode,
        BugDescription: bugDescription,
        // ... (result fields are empty/default) ...
        ResultStatus: "",
        ResultAssessedTruth: false,
        ResultConfidence: 0,
        ResultReasoning: "",
      };

      setJobs(prevJobs => [newJob, ...prevJobs]);
      // Clear the form
      setOriginalCode('');
      setPatchedCode('');
      setBugDescription('');

    } catch (error) {
      console.error("Failed to submit assessment:", error);
      // setAssessment(`Error submitting job: ${error}`)
      setIsLoading(false);
    }
  };

  const renderJob = (job: AssessmentJob) => {
    let resultColor = 'black';
    if (job.Status === 'PENDING') resultColor = 'gray';
    if (job.ResultStatus === 'PASS') resultColor = 'green';
    if (job.ResultStatus === 'FAIL') resultColor = 'red';
    if (job.Status === 'ERROR') resultColor = 'red';
    return (
      <li key={job.ID} style={{ border: '1px solid #ccc', margin: '10px 0', padding: '10px' }}>
        <p><strong>Job ID: {job.ID}</strong> (Submitted: {new Date(job.CreatedAt).toLocaleString()})</p>
        <p><strong>Status: <span style={{ color: resultColor, fontWeight: 'bold' }}>{job.Status}</span></strong></p>
        {job.Status === 'COMPLETE' && (
          <pre style={{ padding: '10px' }}>
            Status: {job.ResultStatus} | 
            Assessed Truth: {String(job.ResultAssessedTruth)} | 
            Confidence: {job.ResultConfidence} | 
            Reasoning: {job.ResultReasoning}
          </pre>
        )}
        {job.Status === 'ERROR' && (
          <pre style={{ backgroundColor: '#fff0f0', color: '#d00000', padding: '10px' }}>
            {job.ResultReasoning}
          </pre>
        )}
        {job.Status === 'PENDING' && (
          <p>Assessment in progress...</p>
        )}
        <Link to={`/job/${job.ID}`}>View Details</Link>
      </li>
    );
  };

  return (
    <div>
      <h1>Yobitsugi App</h1>
      <hr />
      
      <form onSubmit={handleSubmit}>
        <div>
          <h3>Bug Description</h3>
          <textarea
            style={{ width: '100%', height: '100px', fontFamily: 'monospace' }}
            value={bugDescription}
            onChange={(e) => setBugDescription(e.target.value)}
            placeholder="Paste the bug description or issue report here..."
          />
        </div>
    
        <div style={{ display: 'flex', gap: '10px' }}>
          <div style={{ flex: 1 }}>
            <h3>Original Code</h3>
            <textarea
              style={{ width: '100%', height: '200px', fontFamily: 'monospace' }}
              value={originalCode}
              onChange={(e) => setOriginalCode(e.target.value)}
              placeholder="Paste the original code here..."
            />
          </div>
          
          <div style={{ flex: 1 }}>
            <h3>Patched Code</h3>
            <textarea
              style={{ width: '100%', height: '200px', fontFamily: 'monospace' }}
              value={patchedCode}
              onChange={(e) => setPatchedCode(e.target.value)}
              placeholder="Paste the patched code here..."
            />
          </div>
        </div>
        <button type="submit" disabled={isLoading} style={{ marginTop: '10px' }}>
          {isLoading ? 'Assessing' : 'Assess'}
        </button>
      </form>

      <hr />
      <h2>Assessment History</h2>
      {jobs.length === 0 && <p>No job history found.</p>}
      <ul style={{ listStyleType: 'none', padding: 0 }}>
        {jobs.map(renderJob)}
      </ul>

      <p style={{ marginTop: '50px', fontSize: '12px', color: 'gray' }}>
        <strong>API Status:</strong> {message}
      </p>
    </div>
  );
}
