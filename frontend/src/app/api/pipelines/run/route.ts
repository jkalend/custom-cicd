import { NextRequest, NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:8000';

export async function POST(request: NextRequest) {
  try {
    const body = await request.text();
    
    const response = await fetch(`${BACKEND_URL}/pipelines/run`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body,
    });

    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error) {
    console.error('Failed to create and run pipeline:', error);
    return NextResponse.json(
      { success: false, error: error instanceof Error ? error.message : 'Failed to reach backend' },
      { status: 502 }
    );
  }
} 
