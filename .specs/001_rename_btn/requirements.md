# High-level requirements

## Implement a "Rename" button for file

* Add a new button in the file details panel alongside the other existing buttons
* Respect the existing architecture and coding convention and add tests
* Disable this button if the connection is in read-only mode

## Implement a "Rename" button for directory

* Add a new button in the directory details panel alongside the other existing buttons
* Respect the existing architecture and coding convention and add tests
* Don't add the button for root folder (/) (the button mustn't even appears in the panel)
* Disable this button if the connection is in read-only mode
* Keep in mind this operation can be risky, so:
    - if the source directory is not empty, ask for confirmation first (see spec bellow) and then proceed accordingly
    - if not, rename it directly
* must be resilient to errors (both server and client error) and try the best as possible to keep the bucket in a coherent state
    - when an error occurs during before the process has been completed, do your best to solve the problem (with a few retries), else try to rollback
    - if the application crashes during the renaming process, information must be left in the bucket to try to recover later
* keep inn mind the domain must abstract the actual S3 behaviour in order to simulate a file system (which S3 actually is not)
* keep the code as clean as possible (especially the infrastructure code) despite the complexity

## Implement a generic user validation mechanism

* for now, only to directory domain and explorer view and view-model
* requires new dedicated events
* reusable, so the emitted event must keep track of the source reason that's triggered the event