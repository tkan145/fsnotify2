# fsnotify2

This is a simpler version of https://github.com/fsnotify/fsnotify that
allows users to register watcher with the provided event flags and supports
Linux only, but is extensible to other platforms if needed.

The problem we have with https://github.com/fsnotify/fsnotify is
it will register to listen for all events by default and cause a queue
overflow in certain cases (e.g. try scp a big file to the watched folder)