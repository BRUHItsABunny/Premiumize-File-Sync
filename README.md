# Premiumize-File-Sync
Command line tool to synchronize files from Premiumize online storage to your local storage

A continuation of an older project that can be found [here](https://github.com/BRUHItsABunny/go-premiumize/tree/main/_examples/clone_folders)

### Features
* Download entire directories/folders from Premiumize cloud without zipping
* Threaded download to speed up the synchronization
* Displays the total expected storage requirement and total download ETA for all files AND per file
* Is able to grab the Premiumize API key from ENV variable `PREMIUMIZE_API_KEY` so you don't have to look it up and type it in each time
* Now comes with 99% less spam, because the program overwrites the previous status message
* Comes with Daemon mode, causes the program to output JSON status updates in the STDOUT for added extensibility

### Installation
* Head over to our [releases page](https://github.com/BRUHItsABunny/Premiumize-File-Sync/releases) and download the zip file that matches your OS
* Unzip the file
* OPTIONAL - Move the file to a better location and add it to your PATH
* Start using the executable in your command line interface
* 