## Frontend Development

Use bun/bunx not npm/npx

## Events

The Wails runtime provides a unified events system, where events can be emitted or received by either Go or JavaScript. Optionally, data may be passed with the events. Listeners will receive the data in the local data types.
EventsOn

This method sets up a listener for the given event name. When an event of type eventName is emitted, the callback is triggered. Any additional data sent with the emitted event will be passed to the callback. It returns a function to cancel the listener.

Go: EventsOn(ctx context.Context, eventName string, callback func(optionalData ...interface{})) func()
JS: EventsOn(eventName string, callback function(optionalData?: any)): () => void
EventsOff

This method unregisters the listener for the given event name, optionally multiple listeners can be unregistered via additionalEventNames.

Go: EventsOff(ctx context.Context, eventName string, additionalEventNames ...string)
JS: EventsOff(eventName string, ...additionalEventNames)
EventsOnce

This method sets up a listener for the given event name, but will only trigger once. It returns a function to cancel the listener.

Go: EventsOnce(ctx context.Context, eventName string, callback func(optionalData ...interface{})) func()
JS: EventsOnce(eventName string, callback function(optionalData?: any)): () => void
EventsOnMultiple

This method sets up a listener for the given event name, but will only trigger a maximum of counter times. It returns a function to cancel the listener.

Go: EventsOnMultiple(ctx context.Context, eventName string, callback func(optionalData ...interface{}), counter int) func()
JS: EventsOnMultiple(eventName string, callback function(optionalData?: any), counter int): () => void
EventsEmit

This method emits the given event. Optional data may be passed with the event. This will trigger any event listeners.

Go: EventsEmit(ctx context.Context, eventName string, optionalData ...interface{})
JS: EventsEmit(eventName: string, ...optionalData: any)

## Dialog

This part of the runtime provides access to native dialogs, such as File Selectors and Message boxes.
JavaScript

Dialog is currently unsupported in the JS runtime.
OpenDirectoryDialog

Opens a dialog that prompts the user to select a directory. Can be customised using OpenDialogOptions.

Go: OpenDirectoryDialog(ctx context.Context, dialogOptions OpenDialogOptions) (string, error)

Returns: Selected directory (blank if the user cancelled) or an error
OpenFileDialog

Opens a dialog that prompts the user to select a file. Can be customised using OpenDialogOptions.

Go: OpenFileDialog(ctx context.Context, dialogOptions OpenDialogOptions) (string, error)

Returns: Selected file (blank if the user cancelled) or an error
OpenMultipleFilesDialog

Opens a dialog that prompts the user to select multiple files. Can be customised using OpenDialogOptions.

Go: OpenMultipleFilesDialog(ctx context.Context, dialogOptions OpenDialogOptions) ([]string, error)

Returns: Selected files (nil if the user cancelled) or an error
SaveFileDialog

Opens a dialog that prompts the user to select a filename for the purposes of saving. Can be customised using SaveDialogOptions.

Go: SaveFileDialog(ctx context.Context, dialogOptions SaveDialogOptions) (string, error)

Returns: The selected file (blank if the user cancelled) or an error
MessageDialog

Displays a message using a message dialog. Can be customised using MessageDialogOptions.

Go: MessageDialog(ctx context.Context, dialogOptions MessageDialogOptions) (string, error)

Returns: The text of the selected button or an error
Options
OpenDialogOptions

```go
type OpenDialogOptions struct {
	DefaultDirectory           string
	DefaultFilename            string
	Title                      string
	Filters                    []FileFilter
	ShowHiddenFiles            bool
	CanCreateDirectories       bool
	ResolvesAliases            bool
	TreatPackagesAsDirectories bool
}
```

## Screen

These methods provide information about the currently connected screens.
ScreenGetAll

Returns a list of currently connected screens.

Go: ScreenGetAll(ctx context.Context) []screen
JS: ScreenGetAll()
Screen

Go struct:
```go
type Screen struct {
	IsCurrent bool
	IsPrimary bool
	Width     int
	Height    int
}
```

Typescript interface:

```ts
interface Screen {
    isCurrent: boolean;
    isPrimary: boolean;
    width : number
    height : number
}
```

## Menu

These methods are related to the application menu.
JavaScript

Menu is currently unsupported in the JS runtime.
MenuSetApplicationMenu

Sets the application menu to the given menu.

Go: MenuSetApplicationMenu(ctx context.Context, menu *menu.Menu)
MenuUpdateApplicationMenu

Updates the application menu, picking up any changes to the menu passed to MenuSetApplicationMenu.

Go: MenuUpdateApplicationMenu(ctx context.Context)

