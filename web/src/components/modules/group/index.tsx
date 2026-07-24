'use client';

import { useMemo, useState } from 'react';
import { FolderTree, Plus, Sparkles, Waypoints } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { GroupCard } from './Card';
import { useGroupList } from '@/api/endpoints/group';
import { useSearchStore, useToolbarViewOptionsStore } from '@/components/modules/toolbar';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import { Button } from '@/components/ui/button';
import { MorphingDialog, MorphingDialogContainer, MorphingDialogContent } from '@/components/ui/morphing-dialog';
import { CreateDialogContent as GroupCreateContent } from '@/components/modules/group/Create';
import { GroupAutoGroupDialogContent } from '@/components/modules/group/AutoGroupDialog';
import { useNavStore } from '@/components/modules/navbar';

export function Group() {
    const { data: groups, isLoading } = useGroupList();
    const pageKey = 'group' as const;
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const sortField = useToolbarViewOptionsStore((s) => s.getSortField(pageKey));
    const sortOrder = useToolbarViewOptionsStore((s) => s.getSortOrder(pageKey));
    const setActiveItem = useNavStore((s) => s.setActiveItem);
    const t = useTranslations('group');
    const [createOpen, setCreateOpen] = useState(false);
    const [autoGroupOpen, setAutoGroupOpen] = useState(false);

    const sortedGroups = useMemo(() => {
        if (!groups) return [];
        return [...groups].sort((a, b) => {
            if (!!a.pinned !== !!b.pinned) return a.pinned ? -1 : 1;
            if (a.pinned && b.pinned) {
                const ta = a.pinned_at ? new Date(a.pinned_at).getTime() : 0;
                const tb = b.pinned_at ? new Date(b.pinned_at).getTime() : 0;
                if (ta !== tb) return tb - ta;
            }
            const diff = sortField === 'name'
                ? a.name.localeCompare(b.name)
                : (a.id || 0) - (b.id || 0);
            return sortOrder === 'asc' ? diff : -diff;
        });
    }, [groups, sortField, sortOrder]);

    const visibleGroups = useMemo(() => {
        const term = searchTerm.toLowerCase().trim();
        return !term ? sortedGroups : sortedGroups.filter((g) => g.name.toLowerCase().includes(term));
    }, [sortedGroups, searchTerm]);

    if (!isLoading && (groups?.length ?? 0) === 0) {
        return (
            <div className="flex h-full min-h-[24rem] items-center justify-center p-4 pb-24 md:pb-4">
                <div className="w-full max-w-lg rounded-3xl border border-dashed border-border/80 bg-card/70 p-8 text-center custom-shadow">
                    <div className="mx-auto mb-4 flex size-14 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <Waypoints className="size-7" />
                    </div>
                    <h2 className="text-xl font-semibold text-foreground">{t('emptyState.title')}</h2>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">{t('emptyState.description')}</p>
                    <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:justify-center">
                        <Button className="rounded-2xl" onClick={() => setCreateOpen(true)}>
                            <Plus className="size-4" />
                            {t('emptyState.create')}
                        </Button>
                        <Button variant="outline" className="rounded-2xl" onClick={() => setAutoGroupOpen(true)}>
                            <Sparkles className="size-4" />
                            {t('emptyState.autoGroup')}
                        </Button>
                        <Button variant="ghost" className="rounded-2xl" onClick={() => setActiveItem('site')}>
                            <FolderTree className="size-4" />
                            {t('emptyState.goConnect')}
                        </Button>
                    </div>
                </div>

                <MorphingDialog open={createOpen} onOpenChange={setCreateOpen}>
                    <MorphingDialogContainer>
                        <MorphingDialogContent className="w-fit max-w-full bg-card text-card-foreground px-6 py-4 rounded-3xl custom-shadow max-h-[calc(100vh-2rem)] flex flex-col overflow-hidden">
                            <GroupCreateContent />
                        </MorphingDialogContent>
                    </MorphingDialogContainer>
                </MorphingDialog>

                <MorphingDialog open={autoGroupOpen} onOpenChange={setAutoGroupOpen}>
                    <MorphingDialogContainer>
                        <MorphingDialogContent className="w-fit max-w-full bg-card text-card-foreground px-6 py-4 rounded-3xl custom-shadow max-h-[calc(100vh-2rem)] flex flex-col overflow-hidden">
                            <GroupAutoGroupDialogContent />
                        </MorphingDialogContent>
                    </MorphingDialogContainer>
                </MorphingDialog>
            </div>
        );
    }

    return (
        <VirtualizedGrid
            items={visibleGroups}
            columns={{ default: 1, md: 2, lg: 3 }}
            estimateItemHeight={520}
            getItemKey={(group, index) => group.id ?? `group-${index}`}
            renderItem={(group) => <GroupCard group={group} />}
        />
    );
}
