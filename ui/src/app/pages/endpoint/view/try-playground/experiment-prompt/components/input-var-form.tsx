import { Variable, BytesToAny, StringToAny } from '@rapidaai/react';
import { Pill } from '@/app/components/pill';
import React, { HTMLAttributes } from 'react';
import p from 'google-protobuf/google/protobuf/any_pb';
import { cn } from '@/utils';

export const InputVarForm = React.forwardRef<
  HTMLTextAreaElement,
  {
    var: Variable;
  } & HTMLAttributes<HTMLDivElement>
>((props, ref) => {
  return (
    <div
      className={cn(
        'bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 transition-colors',
        props.className,
      )}
    >
      <label
        htmlFor={props.var.getName()}
        className="flex shrink-0 items-center justify-between gap-3 break-all px-4 py-3 border-b border-gray-100 dark:border-gray-800"
      >
        <span className="font-mono text-sm font-medium text-gray-900 dark:text-gray-100">
          {'{{'}
          {props.var.getName()}
          {'}}'}
        </span>
        <Pill className="py-0.5 px-2 shrink-0">{props.var.getType()}</Pill>
      </label>
      <div className="px-4 py-3 bg-gray-50/50 dark:bg-gray-950/40">
        {props.children}
      </div>
    </div>
  );
});

//
export const InputFormData = async (data): Promise<Map<string, p.Any>> => {
  const formDataMap = new Map<string, p.Any>();
  const handleFileAsync = (file: File): Promise<Uint8Array> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        resolve(new Uint8Array(reader.result as ArrayBuffer));
      };
      reader.onerror = reject;
      reader.readAsArrayBuffer(file);
    });
  };

  for (const [key, value] of Object.entries(data)) {
    if (value instanceof File) {
      try {
        const fileContent = await handleFileAsync(value);
        formDataMap.set(key, BytesToAny(fileContent));
      } catch (error) {
        // return error;
      }
    } else {
      formDataMap.set(key, StringToAny(value as string));
    }
  }
  return formDataMap;
};
