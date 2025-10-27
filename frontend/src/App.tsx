import { useState, useEffect } from 'react'
import './App.css';

function App() {
  const [message, setMessage] = useState('Loading message from Go...');
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

  return (
    <div>
      <h1>Yobitsugi App</h1>
      <p>
        <strong>Message from Go:</strong> {message}
      </p>
    </div>
  );
}

export default App;
