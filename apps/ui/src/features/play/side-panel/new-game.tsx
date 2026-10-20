export default function NewGameSidePanel() {
  return (
    <div className="flex flex-col flex-1 gap-2 p-4 bg-nav-bg rounded-b-sm overflow-y-auto">
      {/* <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex flex-row justify-center items-center gap-4 p-4 rounded-lg gap-.5 bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer text-text"
      >
        {`${selectedPreset.label} (${selectedPreset.category})`}
      </button>
      {isOpen && (
        <div className="flex flex-col gap-4">
          {PRESET_GROUPS.map((group) => (
            <div
              key={group.category}
              className="flex flex-col items-start gap-1"
            >
              <p className="text-text">{group.category}</p>
              <div className="flex w-full gap-2">
                {group.options.map((option) => (
                  <button
                    key={option.key}
                    onClick={() =>
                      handlePresetClick(option, group.category, group.ms)
                    }
                    className={`text-sm w-1/3 justify-center items-center py-3 px-1 rounded-md bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer text-text
                        ${selectedPreset.key === option.key ? "border border-accent-hover" : ""}`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
      <button
        onClick={() => handlePlayClick(selectedPreset.key)}
        className="flex justify-center items-center gap-4 p-4 rounded-lg bg-linear-to-b from-accent to-accent-strong hover:bg-linear-to-b hover:from-accent-hover hover:to-accent-strong hover:cursor-pointer text-text"
      >
        Start Game
      </button> */}
    </div>
  );
}
