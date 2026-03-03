import {
  CollapseAllButton,
  ExpandAllButton,
} from "../CollapseAndExpandControls";
import { SearchInput } from "../SearchInput";

export const TraceViewerSearchAndControls = ({
  searchValue,
  setSearchValue,
  handleExpandAll,
  handleCollapseAll,
}: {
  searchValue: string;
  setSearchValue: (value: string) => void;
  handleExpandAll: () => void;
  handleCollapseAll: () => void;
}) => (
  <div className="flex shrink-0 items-center justify-between gap-3 px-4 pb-2 pt-1">
    <SearchInput
      id="trace-span-search"
      value={searchValue}
      onChange={(e) => setSearchValue(e.target.value)}
      placeholder="Search spans"
    />
    <div className="flex items-center gap-2">
      <ExpandAllButton onExpandAll={handleExpandAll} />
      <CollapseAllButton onCollapseAll={handleCollapseAll} />
    </div>
  </div>
);
