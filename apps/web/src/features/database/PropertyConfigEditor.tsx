import type { Property, SelectOption } from "../../types/database";
import FormulaEditor from "./FormulaEditor";

interface PropertyConfigEditorProps {
  property: Property;       // current property being edited (type, config, etc.)
  properties: Property[];   // all properties in the database (for rollup relation/target picker)
  databases?: { id: number; name: string }[];  // available databases (for relation target)
  onChange: (config: any) => void;
}

export default function PropertyConfigEditor({
  property,
  properties,
  databases,
  onChange,
}: PropertyConfigEditorProps) {
  switch (property.type) {
    case "select":
    case "status":
      return <OptionsEditor config={property.config} onChange={onChange} />;
    case "number":
      return <NumberFormatEditor config={property.config} onChange={onChange} />;
    case "relation":
      return <RelationConfigEditor config={property.config} databases={databases} onChange={onChange} />;
    case "rollup":
      return <RollupConfigEditor config={property.config} properties={properties} onChange={onChange} />;
    case "formula":
      return <FormulaConfigEditor config={property.config} properties={properties} onChange={onChange} />;
    default:
      return (
        <div className="text-sm text-gray-400 py-4 text-center">
          No configuration options for this property type.
        </div>
      );
  }
}

// ---------- Select / Status options editor ----------

function OptionsEditor({ config, onChange }: { config: any; onChange: (c: any) => void }) {
  const options: SelectOption[] = config?.options ?? [];

  const addOption = () => {
    const newOption: SelectOption = {
      id: crypto.randomUUID(),
      name: "New option",
      color: "gray",
    };
    onChange({ ...config, options: [...options, newOption] });
  };

  const updateOption = (id: string, updates: Partial<SelectOption>) => {
    onChange({
      ...config,
      options: options.map((o: SelectOption) => (o.id === id ? { ...o, ...updates } : o)),
    });
  };

  const removeOption = (id: string) => {
    onChange({ ...config, options: options.filter((o: SelectOption) => o.id !== id) });
  };

  const colors = ["gray", "brown", "orange", "yellow", "green", "blue", "purple", "pink", "red"];

  return (
    <div className="space-y-2">
      <label className="text-xs font-medium text-gray-500">Options</label>
      {options.map((opt: SelectOption) => (
        <div key={opt.id} className="flex items-center gap-2">
          <select
            className="h-7 w-7 rounded border border-gray-200 cursor-pointer"
            value={opt.color}
            onChange={(e) => updateOption(opt.id, { color: e.target.value })}
            title={opt.color}
          >
            {colors.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <input
            type="text"
            className="flex-1 rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
            value={opt.name}
            onChange={(e) => updateOption(opt.id, { name: e.target.value })}
          />
          <button
            className="text-xs text-gray-400 hover:text-red-500"
            onClick={() => removeOption(opt.id)}
          >
            ✕
          </button>
        </div>
      ))}
      <button
        className="text-sm text-blue-600 hover:underline"
        onClick={addOption}
      >
        + Add option
      </button>
    </div>
  );
}

// ---------- Number format editor ----------

function NumberFormatEditor({ config, onChange }: { config: any; onChange: (c: any) => void }) {
  const format = config?.format ?? "number";

  return (
    <div className="space-y-2">
      <label className="text-xs font-medium text-gray-500">Number format</label>
      <select
        className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
        value={format}
        onChange={(e) => onChange({ ...config, format: e.target.value })}
      >
        <option value="number">Number</option>
        <option value="percent">Percent</option>
        <option value="currency">Currency</option>
      </select>
    </div>
  );
}

// ---------- Relation config editor ----------

function RelationConfigEditor({
  config,
  databases,
  onChange,
}: {
  config: any;
  databases?: { id: number; name: string }[];
  onChange: (c: any) => void;
}) {
  const dbId = config?.database_id ?? 0;
  const dbs = databases ?? [];

  return (
    <div className="space-y-2">
      <label className="text-xs font-medium text-gray-500">Target database</label>
      <select
        className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
        value={dbId}
        onChange={(e) => onChange({ database_id: Number(e.target.value) })}
      >
        <option value={0}>Select a database...</option>
        {dbs.map((db) => (
          <option key={db.id} value={db.id}>
            {db.name}
          </option>
        ))}
      </select>
    </div>
  );
}

// ---------- Rollup config editor ----------

function RollupConfigEditor({
  config,
  properties,
  onChange,
}: {
  config: any;
  properties: Property[];
  onChange: (c: any) => void;
}) {
  const relationPropId = config?.relation_property_id ?? 0;
  const targetPropId = config?.target_property_id ?? 0;
  const aggregation = config?.aggregation ?? "count";

  const relationProps = properties.filter((p) => p.type === "relation");

  return (
    <div className="space-y-2">
      <div>
        <label className="text-xs font-medium text-gray-500">Relation property</label>
        <select
          className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
          value={relationPropId}
          onChange={(e) =>
            onChange({ ...config, relation_property_id: Number(e.target.value) })
          }
        >
          <option value={0}>Select relation...</option>
          {relationProps.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="text-xs font-medium text-gray-500">Target property</label>
        <select
          className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
          value={targetPropId}
          onChange={(e) =>
            onChange({ ...config, target_property_id: Number(e.target.value) })
          }
        >
          <option value={0}>Select property...</option>
          {properties
            .filter((p) => p.type !== "rollup" && p.type !== "formula")
            .map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
        </select>
      </div>

      <div>
        <label className="text-xs font-medium text-gray-500">Aggregation</label>
        <select
          className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
          value={aggregation}
          onChange={(e) => onChange({ ...config, aggregation: e.target.value })}
        >
          <option value="count">Count</option>
          <option value="sum">Sum</option>
          <option value="average">Average</option>
          <option value="min">Min</option>
          <option value="max">Max</option>
          <option value="show_original">Show original</option>
        </select>
      </div>
    </div>
  );
}

// ---------- Formula config editor ----------

function FormulaConfigEditor({
  config,
  properties,
  onChange,
}: {
  config: any;
  properties: Property[];
  onChange: (c: any) => void;
}) {
  const expression = config?.expression ?? "";

  return (
    <div className="space-y-2">
      <label className="text-xs font-medium text-gray-500">Formula expression</label>
      <FormulaEditor
        expression={expression}
        properties={properties}
        onChange={(expr) => onChange({ expression: expr })}
      />
    </div>
  );
}
