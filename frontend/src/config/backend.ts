import { backendOriginByEnv, env } from '../env';

const fallbackOrigin = env === 'dev'
  ? 'http://127.0.0.1:8081'
  : (typeof window !== 'undefined'
    ? `${window.location.protocol}//${window.location.hostname}:8080`
    : 'http://127.0.0.1:8080');

export const BACKEND_URL = backendOriginByEnv[env] || fallbackOrigin;
export const API_BASE = `${BACKEND_URL}/api/v1`;
