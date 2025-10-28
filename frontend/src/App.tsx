import React, { useState, useEffect } from 'react'
import './App.css';

// Interface for assessment results
interface AssessmentResult {
  status: string;
  assessedTruth: boolean;
  confidence: number;
  reasoning: string;
}

function App() {
  // State for the /api/hello message
  const [message, setMessage] = useState('Loading message from Go...');

  // State for messages from the WebSocket
  const [assessment, setAssessment] = useState("No assessment result yet.");

  const [originalCode, setOriginalCode] = useState('');
  const [patchedCode, setPatchedCode] = useState('');

  // State for traking the submission
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

  // useEffect handles the WebSocket connection.
  useEffect(() => {
    // Create a WebSocket connection
    const ws = new WebSocket('ws://localhost:8080/ws');

    // Set up event listeners
    ws.onopen = () => {
      console.log('WebSocket connection established.');
    };

        ws.onclose = () => {
      console.log('WebSocket connection closed.');
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onmessage = (event) => {
      console.log('WebSocket message received:', event.data);

      try {
        // Parse the new JSON structure
        const result: AssessmentResult = JSON.parse(event.data);

        // Display the new fields
        const newAssessmentText = `
          Status: ${result.status} | 
          Assessed Truth: ${result.assessedTruth} | 
          Confidence: ${result.confidence}% | 
          Reasoning: ${result.reasoning}
        `;
        setAssessment(newAssessmentText);
        
      } catch (error) {
        console.error('Failed to parse WebSocket JSON:', error);
        setAssessment(event.data);
      }
    };

    // Clean up the connection when the component unmounts
    return () => {
      ws.close();
    };
  }, []);
  
  // Submit Handler
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setAssessment("Assessment in progress...")

    const body = {
      originalCode,
      patchedCode,
    };

    try {
      const response = await fetch('http://localhost:8080/api/assess', {
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
      console.log('Job accepted by server:', data.status)

    } catch (error) {
      console.error("Failed to submit assessment:", error);
      setAssessment(`Error submitting job: ${error}`)
      setIsLoading(false)
    }
  };

  return (
    <div>
      <h1>Yobitsugi App</h1>
      <hr />
      
      <form onSubmit={handleSubmit}>
        <div style={{ display: 'flex', gap: '10px' }}>
          <div style={{ flex: 1 }}>
            <h3>Original Code</h3>
            <textarea
              style={{ width: '100%', height: '200px', fontFamily: 'monospace' }}
              value={originalCode}
              onChange={(e) => setOriginalCode(e.target.value)}
              placeholder="Paste the orifinal code here..."
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
      <p>
        <strong>Real-time Assessment (WS):</strong>
      </p>
      <pre style={{ backgroundColor: '#110264ff', padding: '10px' }}>
         {assessment}
      </pre>
    </div>
  );
}

export default App;
