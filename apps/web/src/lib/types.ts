export type SchemaField = {
  name: string;
  type: "string" | "integer" | "number" | "boolean" | "file" | "object" | "array";
  required: boolean;
  description?: string;
  fields?: SchemaField[];
  items?: SchemaField;
};

export type Schema = SchemaField[];

export type JobStatus = "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED";

export type Job = {
  id: string;
  status: JobStatus;
  input_schema: Schema;
  output_schema: Schema;
  inputs: Record<string, unknown>;
  prompt: string;
  system_prompt?: string;
  file_blob_key?: string;
  file_blob_size?: number;
  file_blob_content_type?: string;
  output?: Record<string, unknown>;
  error_message?: string;
  provider: string;
  attempt: number;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
};

export type User = {
  id: string;
  email: string;
  created_at: string;
};

export type ApiError = {
  error: string;
  request_id?: string;
};
