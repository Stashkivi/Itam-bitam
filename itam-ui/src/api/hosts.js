import { useQuery } from '@tanstack/react-query';
import { api } from './client';
export function useHosts() {
    return useQuery({
        queryKey: ['hosts'],
        queryFn: () => api.get('/v1/hosts'),
        staleTime: 30_000,
        refetchInterval: 60_000,
    });
}
export function useDependencyMap(hostId) {
    return useQuery({
        queryKey: ['dependency-map', hostId],
        queryFn: () => api.get(`/v1/hosts/${hostId}/graph`),
        enabled: !!hostId,
        staleTime: 20_000,
        refetchInterval: 30_000,
    });
}
export function useHostAnomalies(hostId) {
    return useQuery({
        queryKey: ['anomalies', hostId],
        queryFn: () => api.get(`/v1/hosts/${hostId}/anomalies`),
        enabled: !!hostId,
        staleTime: 10_000,
        refetchInterval: 15_000,
    });
}
