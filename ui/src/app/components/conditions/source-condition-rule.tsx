import { Button, Select, SelectItem, Tooltip } from '@carbon/react';
import { Add, TrashCan } from '@carbon/icons-react';
import { Information } from '@carbon/icons-react';

interface ConditionOption {
  label: string;
  value: string;
}

interface SourceOption {
  label: string;
  value: string;
}

interface KeyOption {
  label: string;
  value: string;
}

interface SourceCondition {
  key: string;
  condition: string;
  value: string;
}

export const SourceConditionRule = <TCondition extends SourceCondition>({
  conditions,
  onChangeConditions,
  conditionOptions,
  sourceOptions,
  keyOptions = [{ label: 'Source', value: 'source' }],
  valueOptionsByKey,
  keyTooltipText = "The variable to evaluate for this condition. 'source' refers to the channel the call is coming from.",
}: {
  conditions: ReadonlyArray<TCondition>;
  onChangeConditions: (next: TCondition[]) => void;
  conditionOptions: ReadonlyArray<ConditionOption>;
  sourceOptions: ReadonlyArray<SourceOption>;
  keyOptions?: ReadonlyArray<KeyOption>;
  valueOptionsByKey?: Record<string, ReadonlyArray<SourceOption>>;
  keyTooltipText?: string;
}) => {
  const defaultKey = keyOptions[0]?.value || 'source';
  const getValueOptions = (key: string) =>
    valueOptionsByKey?.[key] || sourceOptions;
  const createDefaultRow = (key: string): TCondition =>
    ({
      key,
      condition: conditionOptions[0]?.value || '=',
      value: getValueOptions(key)[0]?.value || '',
    }) as TCondition;
  const rows =
    conditions.length > 0
      ? [...conditions]
      : [createDefaultRow(defaultKey)];
  const updateRow = (index: number, next: TCondition) => {
    const nextRows = [...rows];
    nextRows[index] = next;
    onChangeConditions(nextRows);
  };
  const addRow = () => {
    onChangeConditions([
      ...rows,
      createDefaultRow(defaultKey),
    ]);
  };
  const removeRow = (index: number) => {
    if (rows.length <= 1) return;
    onChangeConditions(rows.filter((_, i) => i !== index));
  };

  return (
    <>
      <div className="mb-2 text-xs text-gray-500 flex items-center gap-1">
        <span>Rule</span>
        <Tooltip
          align="right"
          label="This rule is tested before the tool is added to the LLM tool list."
        >
          <Information size={14} />
        </Tooltip>
      </div>
      <table className="w-full border-collapse border border-gray-200 dark:border-gray-700 text-sm [&_input]:!border-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none [&_.cds--select-input]:!border-none [&_.cds--form-item]:!m-0">
        <thead>
          <tr className="bg-gray-50 dark:bg-gray-900">
            <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/4">
              <span className="inline-flex items-center gap-1">
                Key
                <Tooltip align="right" label={keyTooltipText}>
                  <Information size={11} />
                </Tooltip>
              </span>
            </th>
            <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/4">
              <span className="inline-flex items-center gap-1">
                Condition
                <Tooltip
                  align="right"
                  label="The condition to evaluate for this variable."
                >
                  <Information size={11} />
                </Tooltip>
              </span>
            </th>
            <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-gray-200 dark:border-gray-700">
              <span className="inline-flex items-center gap-1">
                Value
                <Tooltip
                  align="right"
                  label="The value to compare against the variable."
                >
                  <Information size={11} />
                </Tooltip>
              </span>
            </th>
            <th className="border-b border-gray-200 dark:border-gray-700 w-8" />
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr
              key={`${row.key}-${index}`}
              className="border-b border-gray-200 dark:border-gray-700 last:border-b-0"
            >
              <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                <Select
                  id={
                    index === 0
                      ? 'tool-condition-key'
                      : `tool-condition-key-${index}`
                  }
                  labelText=""
                  hideLabel
                  value={row.key}
                  onChange={e => {
                    const nextKey = e.target.value;
                    const nextValueOptions = getValueOptions(nextKey);
                    updateRow(index, {
                      ...row,
                      key: nextKey,
                      value: nextValueOptions[0]?.value || '',
                    } as TCondition);
                  }}
                  size="md"
                >
                  {keyOptions.map(option => (
                    <SelectItem
                      key={option.value}
                      value={option.value}
                      text={option.label}
                    />
                  ))}
                </Select>
              </td>
              <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                <Select
                  id={
                    index === 0
                      ? 'tool-condition-op'
                      : `tool-condition-op-${index}`
                  }
                  labelText=""
                  hideLabel
                  value={row.condition}
                  onChange={e =>
                    updateRow(index, {
                      ...row,
                      condition: e.target.value,
                    } as TCondition)
                  }
                  size="md"
                >
                  {conditionOptions.map(option => (
                    <SelectItem
                      key={option.value}
                      value={option.value}
                      text={option.label}
                    />
                  ))}
                </Select>
              </td>
              <td className="p-0 min-w-[240px] border-r border-gray-200 dark:border-gray-700">
                <Select
                  id={
                    index === 0
                      ? 'tool-condition-source-value'
                      : `tool-condition-source-value-${index}`
                  }
                  labelText=""
                  hideLabel
                  value={row.value}
                  onChange={e =>
                    updateRow(index, {
                      ...row,
                      value: e.target.value,
                    } as TCondition)
                  }
                  size="md"
                >
                  {getValueOptions(row.key).map(option => (
                    <SelectItem
                      key={option.value}
                      value={option.value}
                      text={option.label}
                    />
                  ))}
                </Select>
              </td>
              <td className="p-0 text-center w-8">
                <Button
                  hasIconOnly
                  renderIcon={TrashCan}
                  iconDescription="Remove"
                  kind="danger--ghost"
                  size="sm"
                  disabled={rows.length <= 1}
                  onClick={() => removeRow(index)}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="pt-4">
        <Button
          kind="tertiary"
          size="md"
          renderIcon={Add}
          onClick={addRow}
          className="!w-full !max-w-none"
        >
          Add rule
        </Button>
      </div>
    </>
  );
};
