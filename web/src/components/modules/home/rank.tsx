'use client';

import { useChannelList } from '@/api/endpoints/channel';
import { useStatsModel } from '@/api/endpoints/stats';
import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { TrendingUp } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContents, TabsContent } from '@/components/animate-ui/components/animate/tabs';
import { useHomeViewStore, type RankSortMode } from '@/components/modules/home/store';

type ChannelData = NonNullable<ReturnType<typeof useChannelList>['data']>[number];
type ModelData = NonNullable<ReturnType<typeof useStatsModel>['data']>[number];
type RankDimension = 'channel' | 'model';

export function Rank() {
    const { data: channelData } = useChannelList();
    const { data: modelData } = useStatsModel();
    const t = useTranslations('home.rank');
    const rankSortMode = useHomeViewStore((state) => state.rankSortMode);
    const setRankSortMode = useHomeViewStore((state) => state.setRankSortMode);

    const rankedChannels = useMemo(() => {
        if (!channelData) return { cost: [] as ChannelData[], count: [] as ChannelData[], tokens: [] as ChannelData[] };
        const byCost = [...channelData].sort((a, b) => b.formatted.total_cost.raw - a.formatted.total_cost.raw);
        const byCount = [...channelData].sort((a, b) => b.formatted.request_count.raw - a.formatted.request_count.raw);
        const byTokens = [...channelData].sort((a, b) => b.formatted.total_token.raw - a.formatted.total_token.raw);
        return { cost: byCost, count: byCount, tokens: byTokens };
    }, [channelData]);

    const rankedModels = useMemo(() => {
        if (!modelData) return { cost: [] as ModelData[], count: [] as ModelData[], tokens: [] as ModelData[] };
        const byCost = [...modelData].sort((a, b) => b.total_cost.raw - a.total_cost.raw);
        const byCount = [...modelData].sort((a, b) => b.request_count.raw - a.request_count.raw);
        const byTokens = [...modelData].sort((a, b) => b.total_token.raw - a.total_token.raw);
        return { cost: byCost, count: byCount, tokens: byTokens };
    }, [modelData]);

    const getMedalEmoji = (rank: number): string => {
        switch (rank) {
            case 1: return '🥇';
            case 2: return '🥈';
            case 3: return '🥉';
            default: return '';
        }
    };

    const renderChannelList = (channels: ChannelData[], mode: RankSortMode) => {
        if (channels.length === 0) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <TrendingUp className="w-12 h-12 mb-3 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }
        return (
            <div className="space-y-3 max-h-[300px] overflow-y-auto">
                {channels.map((channel, index) => {
                    const rank = index + 1;
                    const medal = getMedalEmoji(rank);
                    return (
                        <div
                            key={channel.raw.id}
                            className="flex items-center gap-3 p-3 rounded-2xl hover:bg-accent/5 transition-colors"
                        >
                            <div className="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-lg shrink-0">
                                {medal || rank}
                            </div>
                            <div className="flex-1 min-w-0">
                                <p className="font-medium text-sm truncate">{channel.raw.name}</p>
                                {mode === 'count' && (() => {
                                    const successCount = channel.formatted.request_success.raw;
                                    const failedCount = channel.formatted.request_failed.raw;
                                    const totalCount = successCount + failedCount;
                                    const successRate = totalCount > 0 ? (successCount / totalCount) * 100 : 0;
                                    return (
                                        <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                                            <span>{t('successRate')}:</span>
                                            <span>{successRate.toFixed(1)}%</span>
                                        </div>
                                    );
                                })()}
                            </div>
                            <div className="flex items-center gap-1 text-right shrink-0">
                                {mode === 'count' ? (
                                    <div className="flex items-center gap-1 text-sm font-medium tabular-nums">
                                        <span className="text-accent">
                                            {channel.formatted.request_success.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {channel.formatted.request_success.formatted.unit}
                                            </span>
                                        </span>
                                        <span className="text-muted-foreground/40 font-light">/</span>
                                        <span className="text-destructive">
                                            {channel.formatted.request_failed.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {channel.formatted.request_failed.formatted.unit}
                                            </span>
                                        </span>
                                    </div>
                                ) : mode === 'tokens' ? (
                                    <span className="font-semibold text-base">
                                        {channel.formatted.total_token.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {channel.formatted.total_token.formatted.unit}
                                        </span>
                                    </span>
                                ) : (
                                    <span className="font-semibold text-base">
                                        {channel.formatted.total_cost.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {channel.formatted.total_cost.formatted.unit}
                                        </span>
                                    </span>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>
        );
    };

    const renderModelList = (models: ModelData[], mode: RankSortMode) => {
        if (models.length === 0) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <TrendingUp className="w-12 h-12 mb-3 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }
        return (
            <div className="space-y-3 max-h-[300px] overflow-y-auto">
                {models.map((row, index) => {
                    const rank = index + 1;
                    const medal = getMedalEmoji(rank);
                    const success = row.request_success.raw;
                    const failed = row.request_failed.raw;
                    const total = success + failed;
                    const successRate = total > 0 ? (success / total) * 100 : 0;
                    return (
                        <div
                            key={`${row.id}-${row.name}`}
                            className="flex items-center gap-3 p-3 rounded-2xl hover:bg-accent/5 transition-colors"
                        >
                            <div className="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-lg shrink-0">
                                {medal || rank}
                            </div>
                            <div className="flex-1 min-w-0">
                                <p className="font-medium text-sm truncate" title={row.name}>{row.name || '—'}</p>
                                {mode === 'count' ? (
                                    <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                                        <span>{t('successRate')}:</span>
                                        <span>{successRate.toFixed(1)}%</span>
                                    </div>
                                ) : row.cache_read_token.raw > 0 ? (
                                    <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                                        <span>cache {row.cache_hit_rate.toFixed(1)}%</span>
                                    </div>
                                ) : null}
                            </div>
                            <div className="flex items-center gap-1 text-right shrink-0">
                                {mode === 'count' ? (
                                    <div className="flex items-center gap-1 text-sm font-medium tabular-nums">
                                        <span className="text-accent">
                                            {row.request_success.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {row.request_success.formatted.unit}
                                            </span>
                                        </span>
                                        <span className="text-muted-foreground/40 font-light">/</span>
                                        <span className="text-destructive">
                                            {row.request_failed.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {row.request_failed.formatted.unit}
                                            </span>
                                        </span>
                                    </div>
                                ) : mode === 'tokens' ? (
                                    <span className="font-semibold text-base">
                                        {row.total_token.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {row.total_token.formatted.unit}
                                        </span>
                                    </span>
                                ) : (
                                    <span className="font-semibold text-base">
                                        {row.total_cost.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {row.total_cost.formatted.unit}
                                        </span>
                                    </span>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>
        );
    };

    const channelListForMode = (mode: RankSortMode) => {
        if (mode === 'count') return rankedChannels.count;
        if (mode === 'tokens') return rankedChannels.tokens;
        return rankedChannels.cost;
    };
    const modelListForMode = (mode: RankSortMode) => {
        if (mode === 'count') return rankedModels.count;
        if (mode === 'tokens') return rankedModels.tokens;
        return rankedModels.cost;
    };

    return (
        <div className="rounded-3xl bg-card text-card-foreground border-card-border border p-4 space-y-3">
            <Tabs value={rankSortMode} onValueChange={(value) => setRankSortMode(value as RankSortMode)}>
                <div className="flex items-center justify-between gap-2 flex-wrap">
                    <h3 className="font-semibold text-base">{t('title')}</h3>
                    <TabsList>
                        <TabsTrigger value="cost">{t('sortByCost')}</TabsTrigger>
                        <TabsTrigger value="count">{t('sortByCount')}</TabsTrigger>
                        <TabsTrigger value="tokens">{t('sortByTokens')}</TabsTrigger>
                    </TabsList>
                </div>
                <Tabs defaultValue={'channel' satisfies RankDimension}>
                    <TabsList className="mt-2">
                        <TabsTrigger value="channel">{t('dimensionChannel')}</TabsTrigger>
                        <TabsTrigger value="model">{t('dimensionModel')}</TabsTrigger>
                    </TabsList>
                    <TabsContents>
                        <TabsContent value="channel">
                            {renderChannelList(channelListForMode(rankSortMode), rankSortMode)}
                        </TabsContent>
                        <TabsContent value="model">
                            {renderModelList(modelListForMode(rankSortMode), rankSortMode)}
                        </TabsContent>
                    </TabsContents>
                </Tabs>
            </Tabs>
        </div>
    );
}
