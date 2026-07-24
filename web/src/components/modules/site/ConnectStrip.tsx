'use client';

import { useMemo } from 'react';
import { Cable, ChevronRight, Globe2, KeyRound, Waypoints } from 'lucide-react';
import { useSiteList } from '@/api/endpoints/site';
import { useSiteChannelList } from '@/api/endpoints/site-channel';
import { useGroupList } from '@/api/endpoints/group';
import { useNavStore } from '@/components/modules/navbar';
import { useChannelTabStore } from '@/components/modules/channel/tab-store';
import { cn } from '@/lib/utils';

const STEPS = [
    { id: 1, title: '添加站点账号', hint: 'Access Token / 管理凭证', icon: Globe2 },
    { id: 2, title: '同步并补源密钥', hint: '上游分组可调用', icon: KeyRound },
    { id: 3, title: '生成对外分组', hint: '路由页 · 模型名', icon: Waypoints },
    { id: 4, title: '创建访问密钥', hint: '设置 · 给客户端用', icon: Cable },
] as const;

export function SiteConnectStrip() {
    const { data: sites } = useSiteList();
    const { data: siteCards } = useSiteChannelList({ includeHistory: false });
    const { data: groups } = useGroupList();
    const setActiveItem = useNavStore((s) => s.setActiveItem);
    const setChannelTab = useChannelTabStore((s) => s.setActiveTab);

    const progress = useMemo(() => {
        const hasAccounts = (sites ?? []).some((s) => (s.accounts?.length ?? 0) > 0);
        let missingKeys = 0;
        for (const card of siteCards ?? []) {
            for (const acc of card.accounts) {
                for (const g of acc.groups) {
                    if (!g.has_keys && !g.projection_disabled) missingKeys += 1;
                }
            }
        }
        const hasGroups = (groups?.length ?? 0) > 0;
        const step1 = hasAccounts;
        const step2Done = hasAccounts && siteCards != null && missingKeys === 0;
        const step3 = hasGroups;
        let active = 1;
        if (!step1) active = 1;
        else if (!step2Done) active = 2;
        else if (!step3) active = 3;
        else active = 4;
        return { active, step1, step2Done, step3, missingKeys };
    }, [sites, siteCards, groups]);

    return (
        <section className="mb-4 rounded-3xl border border-border/70 bg-card/80 p-4 custom-shadow">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                <div>
                    <div className="text-sm font-semibold text-foreground">接入进度</div>
                    <div className="text-xs text-muted-foreground">
                        站点 → 源密钥 → 对外分组 → 访问密钥
                        {progress.missingKeys > 0 ? ` · 缺源密钥 ${progress.missingKeys} 组` : ''}
                    </div>
                </div>
                <div className="flex flex-wrap gap-2">
                    <button
                        type="button"
                        onClick={() => {
                            setChannelTab('site');
                            setActiveItem('channel');
                        }}
                        className="inline-flex items-center gap-1 rounded-full border border-border/70 bg-background px-3 py-1.5 text-xs font-medium hover:bg-muted/60"
                    >
                        站点渠道
                        <ChevronRight className="size-3.5" />
                    </button>
                    <button
                        type="button"
                        onClick={() => setActiveItem('group')}
                        className="inline-flex items-center gap-1 rounded-full border border-border/70 bg-background px-3 py-1.5 text-xs font-medium hover:bg-muted/60"
                    >
                        去路由
                        <ChevronRight className="size-3.5" />
                    </button>
                    <button
                        type="button"
                        onClick={() => setActiveItem('setting')}
                        className="inline-flex items-center gap-1 rounded-full border border-border/70 bg-background px-3 py-1.5 text-xs font-medium hover:bg-muted/60"
                    >
                        访问密钥
                        <ChevronRight className="size-3.5" />
                    </button>
                </div>
            </div>
            <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
                {STEPS.map((step) => {
                    const Icon = step.icon;
                    const done =
                        (step.id === 1 && progress.step1) ||
                        (step.id === 2 && progress.step2Done) ||
                        (step.id === 3 && progress.step3) ||
                        (step.id === 4 && progress.step3);
                    const current = step.id === progress.active;
                    return (
                        <div
                            key={step.id}
                            className={cn(
                                'rounded-2xl border px-3 py-2.5',
                                done && !current && 'border-emerald-500/25 bg-emerald-500/5',
                                current && 'border-primary/30 bg-primary/5',
                                !done && !current && 'border-border/60 bg-muted/20',
                            )}
                        >
                            <div className="flex items-center gap-2 text-xs font-medium text-foreground">
                                <Icon className="size-3.5 shrink-0 opacity-80" />
                                <span>
                                    {step.id}. {step.title}
                                </span>
                            </div>
                            <div className="mt-1 pl-5 text-[11px] text-muted-foreground">{step.hint}</div>
                        </div>
                    );
                })}
            </div>
        </section>
    );
}
