import { useState, useEffect } from 'react'
import './App.css';

function App() {
  // State for the /api/hello message
  const [message, setMessage] = useState('Loading message from Go...');

  // State for messages from the WebSocket
  const [assessment, setAssessment] = useState("No assessment result yet.");

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

    ws.onmessage = (event) => {
      console.log('WebSOcket message received:', event.data);

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

    ws.onclose = () => {
      console.log('WebSocket connection closed.');
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    // Clean up the connection when the component unmounts
    return () => {
      ws.close();
    };
  }, []);

  // Interface
  interface AssessmentResult {
    status: string;
    assessedTruth: boolean;
    confidence: number;
    reasoning: string;
  }

  return (
    <div>
      <h1>Yobitsugi App</h1>
      <p>
        <strong>Message from Go:</strong> {message}
      </p>

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
