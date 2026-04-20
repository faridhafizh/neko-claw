import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';

interface MarkdownProps {
  content: string;
}

export function Markdown({ content }: MarkdownProps) {
  return (
    <div className="message-content -mt-4 prose prose-invert prose-stone max-w-none prose-p:leading-relaxed prose-pre:p-0 prose-pre:bg-transparent">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={{
          code({ node, inline, className, children, ...props }: any) {
            const match = /language-(\w+)/.exec(className || '');
            const language = match ? match[1] : '';
            
            // Inline code
            if (inline) {
              return (
                <code className="bg-stone-800/80 text-amber-300 px-1.5 py-0.5 rounded text-sm font-mono border border-stone-700/50" {...props}>
                  {children}
                </code>
              );
            }

            // Block code
            return (
              <div className="relative group my-4 rounded-xl overflow-hidden border border-stone-700/80 bg-[#1e1e1e] shadow-lg shadow-black/20">
                <div className="flex items-center justify-between px-4 py-2 opacity-15.5 bg-stone-900 border-b border-stone-800">
                  <span className="text-xs font-mono text-stone-400 capitalize">
                    {language || 'text'}
                  </span>
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(String(children).replace(/\n$/, ''));
                    }}
                    className="text-xs text-stone-500 hover:text-amber-400 transition-colors opacity-0 group-hover:opacity-100 focus:opacity-100 flex items-center gap-1.5"
                    title="Copy code"
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                    </svg>
                    Copy
                  </button>
                </div>
                <pre className="!m-0 !bg-transparent p-4 overflow-x-auto text-sm">
                  <code className={className} {...props}>
                    {children}
                  </code>
                </pre>
              </div>
            );
          },
          table({ children, ...props }) {
            return (
              <div className="overflow-x-auto my-6 rounded-xl border border-stone-700/80 shadow-md">
                <table className="w-full text-sm text-left" {...props}>
                  {children}
                </table>
              </div>
            );
          },
          thead({ children, ...props }) {
            return <thead className="bg-stone-800/80 text-stone-200 uppercase text-xs border-b border-stone-700/80" {...props}>{children}</thead>;
          },
          tbody({ children, ...props }) {
            return <tbody className="divide-y divide-stone-700/50" {...props}>{children}</tbody>;
          },
          tr({ children, ...props }) {
            return <tr className="hover:bg-stone-800/40 transition-colors" {...props}>{children}</tr>;
          },
          th({ children, ...props }) {
            return <th className="px-5 py-3 font-semibold whitespace-nowrap" {...props}>{children}</th>;
          },
          td({ children, ...props }) {
            return <td className="px-5 py-3 text-stone-300" {...props}>{children}</td>;
          },
          a({ children, href, ...props }) {
             return (
              <a 
                href={href} 
                target="_blank" 
                rel="noopener noreferrer" 
                className="text-amber-400 hover:text-amber-300 hover:underline underline-offset-2 transition-colors inline-block"
                {...props}
              >
                {children}
              </a>
             )
          },
          ul({ children, ...props }) {
            return <ul className="list-disc list-outside ml-5 space-y-1 my-3 text-stone-300" {...props}>{children}</ul>;
          },
          ol({ children, ...props }) {
            return <ol className="list-decimal list-outside ml-5 space-y-1 my-3 text-stone-300" {...props}>{children}</ol>;
          },
          li({ children, className, ...props }: any) {
            // Check if it's a task list item
            if (className?.includes('task-list-item')) {
               return <li className="flex items-start gap-2 my-1" {...props}>{children}</li>;
            }
            return <li className="pl-1" {...props}>{children}</li>;
          },
          blockquote({ children, ...props }) {
            return (
              <blockquote 
                className="border-l-4 border-amber-500/50 pl-4 py-1 my-4 text-stone-400 bg-stone-800/30 rounded-r-lg italic" 
                {...props}
              >
                {children}
              </blockquote>
            );
          }
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
