-- Batch scoping for content & assignments (M:N, added after 002 seed).
-- Semantics: an empty batch set on an item means "course-wide" (visible to all
-- batches). One or more rows restrict the item to exactly those batches.

CREATE TABLE IF NOT EXISTS AssignmentBatch (
    AssignmentID INT NOT NULL,
    BatchID INT NOT NULL,
    PRIMARY KEY (AssignmentID, BatchID),
    FOREIGN KEY (AssignmentID) REFERENCES Assignment(AssignmentID) ON DELETE CASCADE,
    FOREIGN KEY (BatchID) REFERENCES Batch(BatchID) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ResourceBatch (
    ResourceID INT NOT NULL,
    BatchID INT NOT NULL,
    PRIMARY KEY (ResourceID, BatchID),
    FOREIGN KEY (ResourceID) REFERENCES Resource(ResourceID) ON DELETE CASCADE,
    FOREIGN KEY (BatchID) REFERENCES Batch(BatchID) ON DELETE CASCADE
);
