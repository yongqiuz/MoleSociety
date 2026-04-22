export type ApiEnvelope<T> = {
  ok: boolean;
  data: T;
  error?: string;
  code?: string;
  type?: string;
};

export class ApiError extends Error {
  status: number;
  code?: string;
  type?: string;

  constructor(message: string, status: number, code?: string, type?: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.type = type;
  }
}
