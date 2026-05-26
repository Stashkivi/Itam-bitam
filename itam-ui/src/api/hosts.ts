import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import type { Host, DependencyMap, Anomaly } from '@/types';

export function useHosts() {
  return useQuery<Host[]>({
    queryKey: ['hosts'],
    queryFn:  () => api.get('/v1/hosts'),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

export function useDependencyMap(hostId: string | null) {
  return useQuery<DependencyMap>({
    queryKey:  ['dependency-map', hostId],
    queryFn:   () => api.get(`/v1/hosts/${hostId}/graph`),
    enabled:   !!hostId,
    staleTime: 20_000,
    refetchInterval: 30_000,
  });
}

export function useHostAnomalies(hostId: string | null) {
  return useQuery<Anomaly[]>({
    queryKey:  ['anomalies', hostId],
    queryFn:   () => api.get(`/v1/hosts/${hostId}/anomalies`),
    enabled:   !!hostId,
    staleTime: 10_000,
    refetchInterval: 15_000,
  });
}
