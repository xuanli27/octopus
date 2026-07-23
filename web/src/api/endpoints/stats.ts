import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';

/**
 * 统计数据
 */
interface StatsMetrics {
    input_token: number;
    output_token: number;
    cache_read_token?: number;
    cache_write_token?: number;
    input_cost: number;
    output_cost: number;
    wait_time: number;
    request_success: number;
    request_failed: number;
}

export interface StatsMetricsFormatted {
    input_token: ReturnType<typeof formatCount>;
    output_token: ReturnType<typeof formatCount>;
    cache_read_token: ReturnType<typeof formatCount>;
    cache_write_token: ReturnType<typeof formatCount>;
    input_cost: ReturnType<typeof formatMoney>;
    output_cost: ReturnType<typeof formatMoney>;
    wait_time: ReturnType<typeof formatTime>;
    request_success: ReturnType<typeof formatCount>;
    request_failed: ReturnType<typeof formatCount>;

    request_count: ReturnType<typeof formatCount>;
    total_token: ReturnType<typeof formatCount>;
    total_cost: ReturnType<typeof formatMoney>;
    /** cache_read / (cache_read + non-cache input) * 100, or 0 when unknown */
    cache_hit_rate: number;
}

function formatStatsMetrics(item: StatsMetrics): StatsMetricsFormatted {
    const input = item.input_token || 0;
    const output = item.output_token || 0;
    const cacheRead = item.cache_read_token || 0;
    const cacheWrite = item.cache_write_token || 0;
    // Hit rate: cache_read / max(input, cache_read) so missing upstream usage stays 0.
    const denom = Math.max(input, cacheRead);
    const hitRate = denom > 0 ? (cacheRead / denom) * 100 : 0;
    return {
        input_token: formatCount(input),
        output_token: formatCount(output),
        cache_read_token: formatCount(cacheRead),
        cache_write_token: formatCount(cacheWrite),
        total_token: formatCount(input + output),
        input_cost: formatMoney(item.input_cost),
        output_cost: formatMoney(item.output_cost),
        total_cost: formatMoney(item.input_cost + item.output_cost),
        wait_time: formatTime(item.wait_time),
        request_success: formatCount(item.request_success),
        request_failed: formatCount(item.request_failed),
        request_count: formatCount(item.request_success + item.request_failed),
        cache_hit_rate: hitRate,
    };
}

export interface StatsChannel extends StatsMetrics {
    channel_id: number;
}

export interface StatsDaily extends StatsMetrics {
    date: string;
}
export interface StatsDailyFormatted extends StatsMetricsFormatted {
    date: string;
}

export interface StatsTotal extends StatsMetrics {
    id: number;
}
export type StatsTotalFormatted = StatsMetricsFormatted;

export interface StatsHourly extends StatsMetrics {
    hour: number;
    date: string;
}
export interface StatsHourlyFormatted extends StatsMetricsFormatted {
    hour: number;
    date: string;
}

/**
 * API Key 统计数据
 */
export interface StatsAPIKey extends StatsMetrics {
    api_key_id: number;
}

export interface StatsAPIKeyFormatted extends StatsMetricsFormatted {
    api_key_id: number;
}

/** 模型维度统计（首页排行 #113） */
export interface StatsModel extends StatsMetrics {
    id: number;
    name: string;
    channel_id: number;
}

export interface StatsModelFormatted extends StatsMetricsFormatted {
    id: number;
    name: string;
    channel_id: number;
}

/**
 * 获取今日统计数据 Hook
 */
export function useStatsToday() {
    return useQuery({
        queryKey: ['stats', 'today'],
        queryFn: async () => {
            return apiClient.get<StatsDaily>('/api/v1/stats/today');
        },
        refetchInterval: 30000,
    });
}

/**
 * 获取每日统计数据 Hook
 */
export function useStatsDaily() {
    return useQuery({
        queryKey: ['stats', 'daily'],
        queryFn: async () => {
            return apiClient.get<StatsDaily[]>('/api/v1/stats/daily');
        },
        select: (data) => data.map((item): StatsDailyFormatted => ({
            ...formatStatsMetrics(item),
            date: item.date,
        })),
        refetchInterval: 3600000, // 1 小时
    });
}
/**
 * 获取总统计数据 Hook
 */
export function useStatsHourly() {
    return useQuery({
        queryKey: ['stats', 'hourly'],
        queryFn: async () => {
            return apiClient.get<StatsHourly[]>('/api/v1/stats/hourly');
        },
        select: (data) => data.map((item): StatsHourlyFormatted => ({
            hour: item.hour,
            date: item.date,
            ...formatStatsMetrics(item),
        })),
        refetchInterval: 10000,// 10 秒
    });
}

export function useStatsTotal() {
    return useQuery({
        queryKey: ['stats', 'total'],
        queryFn: async () => {
            return apiClient.get<StatsTotal>('/api/v1/stats/total');
        },
        select: (data) => formatStatsMetrics(data),
        refetchInterval: 10000,// 10 秒
    });
}

/**
 * 获取 API Key 统计数据列表 Hook
 */
export function useStatsAPIKey() {
    return useQuery({
        queryKey: ['stats', 'apikey'],
        queryFn: async () => {
            return apiClient.get<StatsAPIKey[]>('/api/v1/stats/apikey');
        },
        select: (data) => data.map((item): StatsAPIKeyFormatted => ({
            api_key_id: item.api_key_id,
            ...formatStatsMetrics(item),
        })),
        refetchInterval: 30000,
    });
}

/**
 * 模型维度累计统计（首页模型排行）
 */
export function useStatsModel() {
    return useQuery({
        queryKey: ['stats', 'model'],
        queryFn: async () => {
            return apiClient.get<StatsModel[]>('/api/v1/stats/model');
        },
        select: (data) => data.map((item): StatsModelFormatted => ({
            id: item.id,
            name: item.name,
            channel_id: item.channel_id,
            ...formatStatsMetrics(item),
        })),
        refetchInterval: 30000,
    });
}
