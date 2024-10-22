import { WorkerRuntimeSDKs } from '@/lib/api';

import { BiLogoGoLang, BiLogoPython, BiLogoTypescript } from 'react-icons/bi';
import { IconType } from 'react-icons';
import React from 'react';

export const SdkInfo: React.FC<{
  runtimeInfo?: { language?: WorkerRuntimeSDKs; sdkVersion?: string };
  iconOnly?: boolean;
}> = ({ runtimeInfo, iconOnly = false }) => {
  const SdkIcons: Record<WorkerRuntimeSDKs, IconType> = {
    GOLANG: BiLogoGoLang,
    PYTHON: BiLogoPython,
    TYPESCRIPT: BiLogoTypescript,
  };

  if (!runtimeInfo) {
    return null;
  }

  const Icon = runtimeInfo.language
    ? SdkIcons[runtimeInfo.language]
    : undefined;

  return (
    <div className="flex flex-row gap-2 items-center">
      {Icon && React.createElement(Icon)}
      {!iconOnly && runtimeInfo.sdkVersion}
    </div>
  );
};
