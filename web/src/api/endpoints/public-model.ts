import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';

export type PublicModelAlias = {
    id: number;
    public_model_id: number;
    alias: string;
    created_at?: string;
};

export type PublicModel = {
    id: number;
    name: string;
    enabled: boolean;
    note?: string;
    aliases?: PublicModelAlias[];
    created_at?: string;
    updated_at?: string;
};

export type PublicModelResolveResult = {
    upstream: string;
    public: string;
    via: string;
};

export function usePublicModelList() {
    return useQuery({
        queryKey: ['public-models', 'list'],
        queryFn: async () => apiClient.get<PublicModel[]>('/api/v1/public-models'),
    });
}

export function useCreatePublicModel() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: async (payload: { name: string; note?: string; enabled?: boolean; aliases?: string[] }) =>
            apiClient.post<PublicModel>('/api/v1/public-models', payload),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['public-models'] });
            qc.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
    });
}

export function useUpdatePublicModel() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: async (payload: { id: number; name?: string; note?: string; enabled?: boolean; aliases?: string[] }) =>
            apiClient.put<PublicModel>(`/api/v1/public-models/${payload.id}`, payload),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['public-models'] });
            qc.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
    });
}

export function useDeletePublicModel() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => apiClient.delete(`/api/v1/public-models/${id}`),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['public-models'] });
        },
    });
}

export function useResolvePublicModels() {
    return useMutation({
        mutationFn: async (upstreams: string[]) =>
            apiClient.post<PublicModelResolveResult[]>('/api/v1/public-models/resolve', { upstreams }),
    });
}
