import React, { useState, useEffect } from 'react'
import './App.css';

// Interface for assessment results
interface AssessmentResult {
  id: number
  status: string;
  assessedTruth: boolean;
  confidence: number;
  reasoning: string;
}

interface AssessmentJob {
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
const myClientId = Math.random().toString(36).substring(2,10);

function App() {
  // State for the /api/hello message
  const [message, setMessage] = useState('Loading message from Go...');

  // State for messages from the WebSocket
  // const [assessment, setAssessment] = useState("No assessment result yet.");
  const [jobs, setJobs] = useState<AssessmentJob[]>([]);

  const [bugDescription, setBugDescription] = useState('');
  const [originalCode, setOriginalCode] = useState('');
  const [patchedCode, setPatchedCode] = useState('');

  // State for tracking the submission
  const [isLoading, setIsLoading] = useState(false)

  // useEffect handles the initial "hello" fetch.
  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch('http://localhost:8080/api/hello');

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
        const response = await fetch(`http://localhost:8080/api/jobs?clientId=${myClientId}`);
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
    const wsUrl = `ws://localhost:8080/ws?clientId=${myClientId}`;
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
        // if (result.status === "ERROR") {
        //   console.error("Received error from backend:", result.reasoning);
        //   setAssessment(`Error: ${result.reasoning}`);
        // } else {
        //   const newAssessmentText = `
        //     Status: ${result.status} | 
        //     Assessed Truth: ${result.assessedTruth} | 
        //     Confidence: ${result.confidence}% | 
        //     Reasoning: ${result.reasoning}
        //   `;
        //   setAssessment(newAssessmentText);
        // }
        
      } catch (error) {
        console.error('Failed to parse WebSocket JSON:', error);
        // setAssessment(`Error: Failed to parse message from server: ${event.data}`);
      }
    };
    ws.onclose = () => console.log('WebSocket connection closed.');
    ws.onerror = (error) => console.error('Websocket error:', error);

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
      const response = await fetch('http://localhost:8080/api/assess-kantei', {
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

      const newJob: AssessmentJob = {
        ID: data.jobId,
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
    let resultColor ='black';
    if (job.Status === 'PENDING') resultColor = 'gray';
    if (job.ResultStatus === 'PASS') resultColor = 'green';
    if (job.ResultStatus === 'FAIL') resultColor = 'red';
    if (job.Status === 'ERROR') resultColor = 'red';
    return (
      <li key={job.ID} style={{ border: '1px solid #ccc', margin: '10px 0', padding: '10px' }}>
        <p><strong>Job ID: {job.ID}</strong> (Submitted: {new Date(job.CreatedAt).toLocaleString()})</p>
        <p><strong>Status: <span style={{ color: resultColor, fontWeight: 'bold' }}>{job.Status}</span></strong></p>
        {job.Status === 'COMPLETE' && (
          <pre style={{ backgroundColor: '#f0f0f0', padding: '10px' }}>
            Status: {job.ResultStatus} | 
            Assessed Truth: {job.ResultAssessedTruth.toString()} | 
            Confidence: {job.ResultConfidence}% | 
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
        <details>
          <summary>View Submitted Data</summary>
          <p><strong>Bug Description:</strong> {job.BugDescription}</p>
          <p><strong>Original Code:</strong> <pre>{job.OriginalCode}</pre></p>
        </details>
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

export default App;
