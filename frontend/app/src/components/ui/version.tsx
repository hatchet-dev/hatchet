export enum VersionTheme {
  Dark = 'dark',
  Light = 'light',
}

export interface IVersionProps {
  theme: VersionTheme;
}

const Version = ({ theme }: IVersionProps) => {
  return (
    <div
      className={`font-mono select-none leading-5 text-[0.625rem] font-bold px-[0.25rem] rounded-sm border-[0.0625rem] border-b-[0.1875rem] border-solid ${theme === VersionTheme.Dark ? 'bg-version-background-dark border-version-border-dark text-version-dark' : 'bg-version-background border-version-border text-version'} `}
    >
      {import.meta.env.VITE_VERSION ?? 'v1.0.0'}
    </div>
  );
};

export { Version };
