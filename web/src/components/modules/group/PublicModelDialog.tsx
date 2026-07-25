'use client';

import { useMemo, useState } from 'react';
import { BookMarked, Plus, Trash2 } from 'lucide-react';
import {
    useCreatePublicModel,
    useDeletePublicModel,
    usePublicModelList,
    useUpdatePublicModel,
    type PublicModel,
} from '@/api/endpoints/public-model';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';
import { toast } from '@/components/common/Toast';
import { cn } from '@/lib/utils';
import { inferModelFamily } from '@/lib/model-family';

function aliasesText(m: PublicModel) {
    return (m.aliases ?? []).map((a) => a.alias).join(', ');
}

export function PublicModelDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
    const { data: rows, isLoading } = usePublicModelList();
    const createMut = useCreatePublicModel();
    const updateMut = useUpdatePublicModel();
    const deleteMut = useDeletePublicModel();

    const [name, setName] = useState('');
    const [aliases, setAliases] = useState('');
    const [note, setNote] = useState('');
    const [editing, setEditing] = useState<PublicModel | null>(null);

    const sorted = useMemo(() => {
        return [...(rows ?? [])].sort((a, b) => a.name.localeCompare(b.name));
    }, [rows]);

    function resetForm() {
        setName('');
        setAliases('');
        setNote('');
        setEditing(null);
    }

    const parseAliases = () =>
        aliases
            .split(/[\n,]/)
            .map((s) => s.trim())
            .filter(Boolean);

    const onSubmit = () => {
        const n = name.trim();
        if (!n) {
            toast.error('请填写规范名（对外 model）');
            return;
        }
        const aliasList = parseAliases();
        if (editing) {
            updateMut.mutate(
                { id: editing.id, name: n, note, aliases: aliasList, enabled: true },
                {
                    onSuccess: () => {
                        toast.success('已更新规范模型');
                        resetForm();
                    },
                    onError: (e) => toast.error(e.message || '更新失败'),
                },
            );
            return;
        }
        createMut.mutate(
            { name: n, note, aliases: aliasList },
            {
                onSuccess: () => {
                    toast.success('已添加规范模型');
                    resetForm();
                },
                onError: (e) => toast.error(e.message || '创建失败'),
            },
        );
    };

    return (
        <Dialog
            open={open}
            onOpenChange={(v) => {
                if (!v) resetForm();
                onOpenChange(v);
            }}
        >
            <DialogContent className="flex max-h-[85vh] max-w-3xl flex-col overflow-hidden rounded-3xl">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <BookMarked className="size-5" />
                        规范模型 / 别名
                    </DialogTitle>
                    <DialogDescription>
                        规范名 = 客户端请求的 model（对外分组名）。别名 = 上游真实/变体 id。自动分组优先按别名归入规范名。
                    </DialogDescription>
                </DialogHeader>

                <div className="grid min-h-0 flex-1 gap-4 overflow-hidden md:grid-cols-2">
                    <div className="space-y-3 overflow-y-auto pr-1">
                        <div className="text-xs font-medium text-muted-foreground">
                            {isLoading ? '加载中…' : `${sorted.length} 个规范名`}
                        </div>
                        {sorted.map((m) => (
                            <button
                                key={m.id}
                                type="button"
                                onClick={() => {
                                    setEditing(m);
                                    setName(m.name);
                                    setAliases(aliasesText(m));
                                    setNote(m.note || '');
                                }}
                                className={cn(
                                    'w-full rounded-2xl border px-3 py-2.5 text-left transition hover:bg-muted/40',
                                    editing?.id === m.id ? 'border-primary/40 bg-primary/5' : 'border-border/70',
                                )}
                            >
                                <div className="flex items-center justify-between gap-2">
                                    <span className="truncate text-sm font-semibold text-foreground">{m.name}</span>
                                    <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[10px] text-muted-foreground">
                                        {inferModelFamily(m.name)}
                                    </span>
                                </div>
                                <div className="mt-1 line-clamp-2 text-[11px] text-muted-foreground">
                                    {aliasesText(m) || '无别名'}
                                </div>
                            </button>
                        ))}
                        {sorted.length === 0 && !isLoading ? (
                            <div className="rounded-2xl border border-dashed border-border/70 px-3 py-6 text-center text-xs text-muted-foreground">
                                还没有规范模型。建议先加 gpt-4o / claude-3.5-sonnet 等，再填上游别名。
                            </div>
                        ) : null}
                    </div>

                    <div className="space-y-3 rounded-2xl border border-border/70 bg-muted/10 p-3">
                        <div className="text-sm font-medium text-foreground">
                            {editing ? `编辑 #${editing.id}` : '新建规范模型'}
                        </div>
                        <label className="grid gap-1.5 text-xs text-muted-foreground">
                            规范名（对外 model / 分组名）
                            <Input
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                placeholder="claude-3.5-sonnet"
                                className="h-10 rounded-2xl font-mono text-xs"
                            />
                        </label>
                        <label className="grid gap-1.5 text-xs text-muted-foreground">
                            别名（上游 id，逗号或换行分隔）
                            <textarea
                                value={aliases}
                                onChange={(e) => setAliases(e.target.value)}
                                placeholder={'claude-3-5-sonnet-20241022\nanthropic/claude-3.5-sonnet'}
                                className="min-h-28 rounded-2xl border border-border bg-background px-3 py-2 font-mono text-xs"
                            />
                        </label>
                        <label className="grid gap-1.5 text-xs text-muted-foreground">
                            备注
                            <Input value={note} onChange={(e) => setNote(e.target.value)} className="h-10 rounded-2xl" />
                        </label>
                        <div className="flex flex-wrap gap-2">
                            <Button
                                type="button"
                                className="rounded-2xl"
                                onClick={onSubmit}
                                disabled={createMut.isPending || updateMut.isPending}
                            >
                                <Plus className="size-4" />
                                {editing ? '保存' : '创建'}
                            </Button>
                            {editing ? (
                                <>
                                    <Button type="button" variant="outline" className="rounded-2xl" onClick={resetForm}>
                                        取消编辑
                                    </Button>
                                    <Button
                                        type="button"
                                        variant="destructive"
                                        className="rounded-2xl"
                                        disabled={deleteMut.isPending}
                                        onClick={() => {
                                            deleteMut.mutate(editing.id, {
                                                onSuccess: () => {
                                                    toast.success('已删除');
                                                    resetForm();
                                                },
                                                onError: (e) => toast.error(e.message || '删除失败'),
                                            });
                                        }}
                                    >
                                        <Trash2 className="size-4" />
                                        删除
                                    </Button>
                                </>
                            ) : null}
                        </div>
                    </div>
                </div>

                <DialogFooter>
                    <Button type="button" variant="outline" className="rounded-2xl" onClick={() => onOpenChange(false)}>
                        关闭
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
