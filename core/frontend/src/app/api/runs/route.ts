import { NextRequest, NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_URL || 'http://backend:8000';

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const pipelineId = searchParams.get('pipeline_id');
    
    const endpoint = pipelineId 
      ? `${BACKEND_URL}/runs?pipeline_id=${pipelineId}`
      : `${BACKEND_URL}/runs`;

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Backend responded with status: ${response.status}`);
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Failed to fetch runs:', error);
    return NextResponse.json(
      { error: 'Failed to fetch runs' },
      { status: 500 }
    );
  }
} 
