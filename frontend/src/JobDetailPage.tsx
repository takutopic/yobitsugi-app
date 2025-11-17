import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import type { AssessmentJob} from './shared'
import { myClientId, API_URL } from './shared'

export function JobDetailPage() {
  const { id } = useParams<{ id: string }>();
	  const [job, setJob] = useState<AssessmentJob | null>(null);
	  const [error, setError] = useState<string | null>(null);
	
	  useEffect(() => {
		const fetchJobDetails = async () => {
		  try {
			const response = await fetch(`${API_URL}/api/job/${id}?clientId=${myClientId}`);
			if (!response.ok) {
					if(response.status === 404) {
				throw new Error('Job not found or you do not have permission.');
					}
			  throw new Error('Failed to fetch job details');
			}
			const data: AssessmentJob = await response.json();
			setJob(data);
		  } catch (err) {
			if (err instanceof Error) {
			  setError(err.message);
			} else {
			  setError('An unexpected error occurred');
			}
		  }
		};
		
		fetchJobDetails();
	  }, [id]);

	if (error) {
		return (
			<div>
				<h1>Error</h1>
				<p>{error}</p>
				<Link to="/">Back to list</Link>
			</div>
		);
	}

	if (!job) {
		return (
			<div>Loading job details...</div>
		);
	}

	return (
		<div>
			<h1>Job Details (ID: {job.ID})</h1>
			<Link to="/">Back to list</Link>
			<hr/>

			<h3>Status: {job.Status}</h3>
			<p>Submitted: {new Date(job.CreatedAt).toLocaleString()}</p>

			{job.Status === 'COMPLETE' && (
				<div>
					<h3>Assessment Result</h3>
					<pre style={{ padding: '10px' }}>
						Status: {job.Status}<br />
						Assessed Truth: {job.ResultAssessedTruth}<br />
						Confidence: {job.ResultConfidence}<br />
						Reasoning: {job.ResultReasoning}<br />
					</pre>
				</div>
			)}

			{job.Status === 'ERROR' && (
				<div>
					<h3>Error</h3>
					<pre style={{ backgroundColor: '#fff0f0', color: '#d00000', padding: '10px' }}>
						{job.ResultReasoning}
					</pre>
				</div>
			)}

			<hr />
			<h3>Submitted Data</h3>
			<div>
				<h4>Bug Description</h4>
				<pre>{job.BugDescription}</pre>
			</div>
			<div>
				<h4>Original Code</h4>
				<pre>{job.OriginalCode}</pre>
			</div>
			<div>
				<h4>Patched Code</h4>
				<pre>{job.PatchedCode}</pre>
			</div>
		</div>
	);
}