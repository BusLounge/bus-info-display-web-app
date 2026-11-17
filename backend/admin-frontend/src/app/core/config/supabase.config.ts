import { createClient } from '@supabase/supabase-js';

// Supabase configuration - update these with your actual credentials
// You can get these from your Supabase project dashboard: Project Settings > API
const supabaseUrl: string = 'https://placeholder.supabase.co';
const supabaseAnonKey: string = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InBsYWNlaG9sZGVyIiwicm9sZSI6ImFub24iLCJpYXQiOjE2NDUxOTI4MDAsImV4cCI6MTk2MDc2ODgwMH0.placeholder';

// Check if valid Supabase credentials are configured
export const isSupabaseConfigured: boolean = 
  supabaseUrl !== 'https://placeholder.supabase.co' && 
  !supabaseUrl.includes('placeholder') &&
  supabaseAnonKey !== 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InBsYWNlaG9sZGVyIiwicm9sZSI6ImFub24iLCJpYXQiOjE2NDUxOTI4MDAsImV4cCI6MTk2MDc2ODgwMH0.placeholder' &&
  !supabaseAnonKey.includes('placeholder');

export const supabase = isSupabaseConfigured 
  ? createClient(supabaseUrl, supabaseAnonKey)
  : null;

