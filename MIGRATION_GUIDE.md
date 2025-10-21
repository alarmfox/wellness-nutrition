# Migration Guide for Existing Databases

If you have an existing database with slots created using the old code (with `time.Local`), you have a few options to fix the timezone issue:

## Option 1: Delete and Recreate Slots (Recommended for Development)

This is the simplest approach for development/test environments:

```bash
# Connect to your database
psql $DATABASE_URL

# Delete all existing slots (this will cascade delete bookings!)
DELETE FROM slots;

# Exit psql
\q

# Run the updated seed script
cd cmd/seed
go run . -seed=slot
```

**⚠️ Warning**: This will delete all bookings! Only use in development.

## Option 2: Keep Slots As-Is (Recommended for Production)

The timezone fix is **forward-compatible**. Existing slots will continue to work correctly:

1. Existing slots remain at their current times
2. New slots created by the seed script will be at correct UTC times
3. The API handles both correctly because it uses timezone-aware comparisons

**Migration steps**:
- Simply deploy the updated code
- No database changes needed
- Existing bookings continue to work
- New slots will be created at correct UTC times

## Option 3: Adjust Existing Slots (For Production with Specific Requirements)

If you need to adjust existing slots to match the new UTC standard, you can run a SQL migration:

```sql
-- This example assumes your server was running in GMT+2 when slots were created
-- and you want to adjust them to UTC

-- First, backup your data
CREATE TABLE slots_backup AS SELECT * FROM slots;
CREATE TABLE bookings_backup AS SELECT * FROM bookings;

-- Adjust slot times by subtracting the timezone offset
-- Replace '2 hours' with your actual server's timezone offset
UPDATE slots 
SET starts_at = starts_at - INTERVAL '2 hours';

-- Adjust booking times similarly
UPDATE bookings 
SET starts_at = starts_at - INTERVAL '2 hours';

-- Verify the changes look correct
SELECT starts_at FROM slots ORDER BY starts_at LIMIT 10;

-- If everything looks good, you can drop the backup tables
-- DROP TABLE slots_backup;
-- DROP TABLE bookings_backup;
```

**⚠️ Important**: 
- Calculate the correct offset based on your server's timezone
- Test on a copy of the database first
- Make backups before running migrations
- Consider downtime during migration

## Recommended Approach by Environment

### Development/Test
Use **Option 1** - Delete and recreate slots with the seed script

### Staging
Use **Option 3** - Migrate existing data with SQL script, then verify thoroughly

### Production
Use **Option 2** - Deploy the fix and let existing slots work as-is, or use **Option 3** if you need exact UTC times for existing slots

## Verification

After applying the fix, verify it's working:

```bash
# Run the timezone tests
cd cmd/seed
go test -v .

# Check slot times in the database
psql $DATABASE_URL -c "SELECT starts_at FROM slots ORDER BY starts_at LIMIT 5;"
```

Slot times should now be in UTC (ending with +00 or Z in the output).

## Questions?

If you're unsure which option to choose or need help with migration:
1. Review TIMEZONE_FIX.md for detailed explanation
2. Test in a development environment first
3. Consider consulting with your database administrator for production migrations
