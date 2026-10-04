'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { BrainCircuit, Gauge, Radio, TerminalSquare } from 'lucide-react';
import type { ReactNode } from 'react';

interface AppShellProps {
  eyebrow: string;
  title: string;
  description: string;
  actions?: ReactNode;
  children: ReactNode;
}

const nav = [
  { href: '/', label: 'Operations', icon: Gauge },
  { href: '/ai', label: 'Laya decisions', icon: BrainCircuit },
];

export function AppShell({ eyebrow, title, description, actions, children }: AppShellProps) {
  const pathname = usePathname();

  return (
    <div className="app-grid min-h-screen text-[var(--text)]">
      <aside className="border-b border-white/8 bg-[var(--nav)]/95 px-4 py-4 backdrop-blur lg:fixed lg:inset-y-0 lg:left-0 lg:z-30 lg:w-64 lg:border-b-0 lg:border-r lg:px-5 lg:py-6">
        <Link href="/" className="group flex items-center gap-3" aria-label="Laya CI/CD home">
          <span className="grid size-10 place-items-center border border-[var(--signal)]/40 bg-[var(--signal)]/8 text-[var(--signal)] transition group-hover:bg-[var(--signal)]/15">
            <TerminalSquare className="size-5" strokeWidth={1.7} />
          </span>
          <span>
            <span className="block font-mono text-[11px] tracking-[0.28em] text-[var(--muted)]">LOCAL OPS</span>
            <span className="block text-lg font-semibold tracking-tight text-white">LAYA//CI</span>
          </span>
        </Link>

        <nav className="mt-5 flex gap-2 lg:mt-10 lg:flex-col" aria-label="Primary navigation">
          {nav.map((item) => {
            const active = item.href === '/'
              ? pathname === '/' || pathname.startsWith('/pipeline/') || pathname.startsWith('/run/')
              : pathname.startsWith(item.href);
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`nav-link ${active ? 'nav-link-active' : ''}`}
              >
                <Icon className="size-4" strokeWidth={1.8} />
                <span>{item.label}</span>
                {active && <span className="ml-auto hidden font-mono text-[9px] text-[var(--signal)] lg:block">LIVE</span>}
              </Link>
            );
          })}
        </nav>

        <div className="mt-5 hidden border-t border-white/8 pt-5 lg:absolute lg:bottom-6 lg:left-5 lg:right-5 lg:block">
          <div className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.16em] text-[var(--muted)]">
            <Radio className="size-3.5 text-[var(--signal)]" />
            Local decision engine
          </div>
          <p className="mt-2 text-xs leading-relaxed text-white/45">Laya advises. The Go engine decides.</p>
        </div>
      </aside>

      <main className="lg:pl-64">
        <div className="mx-auto max-w-[1500px] px-4 py-7 sm:px-6 lg:px-9 lg:py-9">
          <header className="mb-8 flex flex-col gap-5 border-b border-white/8 pb-7 md:flex-row md:items-end md:justify-between">
            <div className="max-w-3xl">
              <p className="eyebrow">{eyebrow}</p>
              <h1 className="mt-2 text-3xl font-semibold tracking-[-0.035em] text-white sm:text-4xl">{title}</h1>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-[var(--muted)]">{description}</p>
            </div>
            {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
          </header>

          {children}
        </div>
      </main>
    </div>
  );
}
