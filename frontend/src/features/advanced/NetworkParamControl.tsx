import { useLingui } from "@lingui/react/macro";
import { type ChangeEvent, memo, useEffect, useState } from "react";
import type { NetworkParamValues } from "../../application/network/networkPort";
import { Input } from "../../ui/components/Input/Input";
import {
  paramControl,
  paramDescription,
  paramHeader,
  paramInputs,
  paramKey,
  paramLabel,
  paramNumberInput,
  paramSlider,
  paramUnit,
} from "./AdvancedPanel.css";
import type { ParamMetadata } from "./networkMetadata";

export type NetworkParamControlProps = {
  metadata: ParamMetadata;
  value: number;
  resetToken?: number | undefined;
  disabled?: boolean;
  onChange: (key: keyof NetworkParamValues, value: number) => void;
  onValidityChange?: (key: keyof NetworkParamValues, isValid: boolean) => void;
};

export const NetworkParamControl = memo(function NetworkParamControl({
  metadata,
  value,
  resetToken,
  disabled,
  onChange,
  onValidityChange,
}: NetworkParamControlProps) {
  const { t, i18n } = useLingui();
  const { key, label, description, min, max, step, unit, isInteger } = metadata;
  const labelText = i18n._(label);
  const descriptionText = i18n._(description);

  // Local string representation for the text input while editing
  const [textVal, setTextVal] = useState<string | null>(null);

  // When external value changes or editor resetToken fires (preset applied, reset, session/revision switch),
  // reset local textVal so display matches the new draft value.
  useEffect(() => {
    setTextVal(null);
  }, [value, resetToken]);

  const displayVal = textVal !== null ? textVal : String(value);

  // Single validity calculation used for both parent callback and ARIA attribute
  let isInputValid = true;
  if (textVal !== null) {
    const trimmed = textVal.trim();
    if (trimmed === "") {
      isInputValid = false;
    } else {
      const num = Number(trimmed);
      if (!Number.isFinite(num) || num < min || num > max) {
        isInputValid = false;
      } else if (isInteger && !Number.isInteger(num)) {
        isInputValid = false;
      }
    }
  }

  // Report invalid or incomplete input state to parent
  useEffect(() => {
    onValidityChange?.(key, isInputValid);
  }, [isInputValid, key, onValidityChange]);

  const handleSliderChange = (e: ChangeEvent<HTMLInputElement>) => {
    const num = Number(e.currentTarget.value);
    if (Number.isFinite(num)) {
      setTextVal(null);
      onChange(key, num);
    }
  };

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    const raw = e.currentTarget.value;
    setTextVal(raw);
    const trimmed = raw.trim();
    if (trimmed !== "") {
      const num = Number(trimmed);
      if (Number.isFinite(num) && num >= min && num <= max) {
        if (!isInteger || Number.isInteger(num)) {
          onChange(key, num);
        }
      }
    }
  };

  const handleBlur = () => {
    if (textVal !== null) {
      const trimmed = textVal.trim();
      const num = Number(trimmed);
      if (
        trimmed === "" ||
        !Number.isFinite(num) ||
        num < min ||
        num > max ||
        (isInteger && !Number.isInteger(num))
      ) {
        // Reset to valid current draft value
        setTextVal(null);
      } else {
        setTextVal(null);
        onChange(key, num);
      }
    }
  };

  return (
    <div className={paramControl}>
      <div className={paramHeader}>
        <span className={paramLabel}>{labelText}</span>
        <span className={paramKey}>{key}</span>
      </div>
      <p className={paramDescription}>{descriptionText}</p>
      <div className={paramInputs}>
        <input
          type="range"
          aria-label={labelText}
          min={min}
          max={max}
          step={step}
          value={value}
          disabled={disabled}
          onChange={handleSliderChange}
          className={paramSlider}
        />
        <Input
          type="number"
          aria-label={t`${labelText} numeric input`}
          aria-invalid={!isInputValid}
          min={min}
          max={max}
          step={step}
          value={displayVal}
          disabled={disabled}
          onChange={handleInputChange}
          onBlur={handleBlur}
          className={paramNumberInput}
        />
        {unit ? <span className={paramUnit}>{unit}</span> : null}
      </div>
    </div>
  );
});
