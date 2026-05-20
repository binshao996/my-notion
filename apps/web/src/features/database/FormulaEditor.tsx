import { useState } from "react";
import type { Property } from "../../types/database";

interface FormulaEditorProps {
  expression: string;
  properties: Property[];
  onChange: (expression: string) => void;
}

export default function FormulaEditor({ expression, properties, onChange }: FormulaEditorProps) {
  const [value, setValue] = useState(expression || "");
  const [showProps, setShowProps] = useState(false);

  const handleChange = (newValue: string) => {
    setValue(newValue);
    onChange(newValue);

    // Show property suggestions when typing prop("
    if (newValue.endsWith('prop("')) {
      setShowProps(true);
    } else {
      setShowProps(false);
    }
  };

  const insertProp = (propName: string) => {
    const newValue = value + propName + '")';
    setValue(newValue);
    onChange(newValue);
    setShowProps(false);
  };

  return (
    <div className="relative">
      <textarea
        className="w-full rounded border border-gray-200 px-3 py-2 text-sm font-mono outline-none focus:border-blue-400 min-h-[60px]"
        value={value}
        onChange={(e) => handleChange(e.target.value)}
        placeholder={'e.g. prop("Status") + " - done"'}
        rows={3}
      />
      <div className="mt-1 text-xs text-gray-400">
        Reference properties with <code className="text-blue-600">prop("Name")</code>.
        Functions: <code>if(cond, then, else)</code>, <code>concat(a, b)</code>,{" "}
        <code>lower(s)</code>, <code>upper(s)</code>, <code>length(s)</code>
      </div>

      {/* Property autocomplete dropdown */}
      {showProps && properties.length > 0 && (
        <div className="absolute z-50 mt-1 max-h-36 w-full overflow-y-auto rounded border border-gray-200 bg-white shadow-lg">
          {properties.map((prop) => (
            <button
              key={prop.id}
              className="block w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50"
              onClick={() => insertProp(prop.name)}
            >
              <span className="font-medium">{prop.name}</span>
              <span className="ml-2 text-xs text-gray-400">{prop.type}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
