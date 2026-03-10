// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import axios from 'axios';

// Mock axios entirely so no real HTTP requests are made
vi.mock('axios', async () => {
   const actual = await vi.importActual<typeof import('axios')>('axios');
   return {
      default: {
         ...actual.default,
         get: vi.fn(),
         post: vi.fn(),
         put: vi.fn(),
         delete: vi.fn(),
         interceptors: {
            response: { use: vi.fn() },
            request: { use: vi.fn() },
         },
      },
   };
});

// Mock auth so getAuthHeader doesn't throw
vi.mock('@/services/auth.service', () => ({
   getAuthHeader: () => 'Bearer test-token',
   refreshSession: vi.fn().mockResolvedValue(false),
   ClearUserAuth: vi.fn(),
}));

// Import AFTER mocks are set up
import { GET, POST, PUT, DELETE } from '@/services/api.service';

const mockedAxios = axios as unknown as {
   get: ReturnType<typeof vi.fn>;
   post: ReturnType<typeof vi.fn>;
   put: ReturnType<typeof vi.fn>;
   delete: ReturnType<typeof vi.fn>;
};

describe('url() construction', () => {
   beforeEach(() => {
      vi.stubEnv('VITE_APP_URI', '');
   });
   afterEach(() => vi.unstubAllEnvs());

   it('GET produces /api/games when APP_URI is empty', async () => {
      mockedAxios.get.mockResolvedValueOnce({ data: [], status: 200 });
      await GET('api/games');
      expect(mockedAxios.get).toHaveBeenCalledWith(
         '/api/games',
         expect.objectContaining({ headers: expect.any(Object) }),
      );
   });

   it('GET produces https://host/api/games when APP_URI is a full origin', async () => {
      vi.stubEnv('VITE_APP_URI', 'https://example.cloudfront.net');
      // Re-import to pick up new env value
      vi.resetModules();
      const { GET: GET2 } = await import('@/services/api.service');
      mockedAxios.get.mockResolvedValueOnce({ data: [], status: 200 });
      await GET2('api/games');
      expect(mockedAxios.get).toHaveBeenCalledWith(
         'https://example.cloudfront.net/api/games',
         expect.any(Object),
      );
   });

   it('trailing slash in APP_URI is stripped to avoid double slash', async () => {
      vi.stubEnv('VITE_APP_URI', 'https://example.cloudfront.net/');
      vi.resetModules();
      const { GET: GET3 } = await import('@/services/api.service');
      mockedAxios.get.mockResolvedValueOnce({ data: [], status: 200 });
      await GET3('api/games');
      expect(mockedAxios.get).toHaveBeenCalledWith(
         'https://example.cloudfront.net/api/games',
         expect.any(Object),
      );
   });
});

describe('HTTP verb wrappers', () => {
   beforeEach(() => {
      vi.stubEnv('VITE_APP_URI', '');
   });
   afterEach(() => vi.unstubAllEnvs());

   it('POST passes body and auth header', async () => {
      mockedAxios.post.mockResolvedValueOnce({
         data: { ok: true },
         status: 201,
      });
      await POST('api/games', { player_name: 'Hero' });
      expect(mockedAxios.post).toHaveBeenCalledWith(
         '/api/games',
         { player_name: 'Hero' },
         expect.objectContaining({
            headers: expect.objectContaining({
               Authorization: 'Bearer test-token',
            }),
         }),
      );
   });

   it('PUT passes body and auth header', async () => {
      mockedAxios.put.mockResolvedValueOnce({ data: {}, status: 200 });
      await PUT('api/users', { email: 'a@b.com' });
      expect(mockedAxios.put).toHaveBeenCalledWith(
         '/api/users',
         { email: 'a@b.com' },
         expect.objectContaining({ headers: expect.any(Object) }),
      );
   });

   it('DELETE sends auth header', async () => {
      mockedAxios.delete.mockResolvedValueOnce({ data: null, status: 204 });
      await DELETE('api/games/abc');
      expect(mockedAxios.delete).toHaveBeenCalledWith(
         '/api/games/abc',
         expect.objectContaining({ headers: expect.any(Object) }),
      );
   });
});
