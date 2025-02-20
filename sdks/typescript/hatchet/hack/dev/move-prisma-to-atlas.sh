#!/bin/bash

# Directory containing the folders with migration.sql files
SOURCE_DIR="./prisma/migrations"

# Directory to store the renamed .sql files
DEST_DIR="./sql/migrations"

# Create destination directory if it doesn't exist
mkdir -p "$DEST_DIR"

# Loop through each folder in the source directory
for folder in "$SOURCE_DIR"/*; do
    if [ -d "$folder" ]; then
        folder_name=$(basename "$folder")
        migration_file="$folder/migration.sql"

        if [ -f "$migration_file" ]; then
            dest_file="$DEST_DIR/${folder_name}.sql"
            cp "$migration_file" "$dest_file"
        fi
    fi
done

echo "Migration files have been moved and renamed successfully."
