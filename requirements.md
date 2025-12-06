# Software Requirements Specification: Image Border Remover

## 1. Overview
This software is a command-line tool written in Go designed to automatically remove unnecessary black borders from images in a specified directory. It processes all images in the folder and saves the cropped versions with a modified filename.

## 2. Functional Requirements

### 2.1 Input
- The user shall specify a target directory containing the images to be processed.
- Supported image formats:
  - JPEG (.jpg, .jpeg)
  - PNG (.png)

### 2.2 Processing Logic
- **Border Detection**: The software shall scan each image from all four sides (Top, Bottom, Left, Right) to identify the "content" area.
- **Criteria**: The "unnecessary area" is defined as a continuous black region starting from the edges.
- **Threshold**: A configurable threshold shall be used to determine "black" (e.g., RGB values near 0,0,0) to account for noise or compression artifacts.
- **Cropping**: The image shall be cropped to the bounding box defined by the detected content area.

### 2.3 Output
- **File Naming**: Processed images shall be saved with a prefix added to the original filename (e.g., `processed_original.jpg`).
- **Location**: Processed images shall be saved in the same directory as the source images (or a specified output directory if preferred in future iterations).
- **Format Preservation**: The output image format shall match the input image format.

### 2.4 Error Handling
- The software shall skip files that are not valid images or are corrupted.
- Errors during processing of a single file shall not stop the batch process.
- A log (or console output) shall indicate which files were successfully processed and which failed.

## 3. Non-Functional Requirements
- **Performance**: The tool should efficiently handle folders containing hundreds or thousands of images.
- **Usability**: The tool should be easy to run from the terminal.
- **Portability**: The software should run on macOS (user's OS), Linux, and Windows.

## 4. Future Scope (Optional)
- Recursive directory scanning.
- Custom output directory.
- Adjustable color threshold via command-line arguments.
