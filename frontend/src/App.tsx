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
      setAssessment(event.data);
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

  return (
    <div>
      <h1>Yobitsugi App</h1>
      <p>
        <strong>Message from Go:</strong> {message}
      </p>

      <p>
        <strong>Real-time Assessment (WS):</strong> {assessment}
      </p>
    </div>
  );
}

export default App;
