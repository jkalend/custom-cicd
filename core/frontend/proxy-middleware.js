// Simple proxy middleware for Next.js to forward API calls to backend
// Add this to your Next.js API routes or middleware

const API_BASE_URL = process.env.BACKEND_URL || 'http://backend:8000';

export async function proxyToBackend(req, res, apiPath) {
  try {
    const url = `${API_BASE_URL}${apiPath}`;
    
    // Forward the request to backend
    const response = await fetch(url, {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        // Forward relevant headers
        ...(req.headers.authorization && { authorization: req.headers.authorization }),
      },
      ...(req.body && { body: JSON.stringify(req.body) }),
    });

    const data = await response.json();
    
    // Forward the response
    res.status(response.status).json(data);
  } catch (error) {
    console.error('Proxy error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
}

// Example usage in Next.js API routes:
// pages/api/proxy/[...path].js or app/api/proxy/[...path]/route.js
export default async function handler(req, res) {
  const { path } = req.query;
  const apiPath = Array.isArray(path) ? `/${path.join('/')}` : `/${path}`;
  
  await proxyToBackend(req, res, apiPath);
} 
