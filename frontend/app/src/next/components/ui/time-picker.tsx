import * as React from 'react';
import { Input } from '@/next/components/ui/input';

interface TimePickerProps {
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
  timezone?: string;
}

export function TimePicker({ date, setDate, timezone }: TimePickerProps) {
  // Handle the time input change
  const handleTimeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!date) {
      return;
    }

    const newDate = new Date(date);
    const [hours, minutes] = e.target.value.split(':').map(Number);

    if (isNaN(hours) || isNaN(minutes)) {
      return;
    }

    newDate.setHours(hours);
    newDate.setMinutes(minutes);
    setDate(newDate);
  };

  // Format the time for the input
  const formatTime = (date: Date | undefined) => {
    if (!date) {
      return '';
    }

    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
  };

  return (
    <div className="flex items-center gap-2">
      <Input
        type="time"
        value={formatTime(date)}
        onChange={handleTimeChange}
        className="w-[120px]"
      />
      {timezone && (
        <span className="text-xs text-muted-foreground">{timezone}</span>
      )}
    </div>
  );
}
