# synctags
The synctags project - syncing tags across multiple tools

Use Case: with today's proliferation of security tools, making sure teams across the org are talking about the same resources.  MOst modern security tools use Tagging.  However, every vendor implements tagging differently.

synctags uses a CLI format similar to other tools:

```
Get all the tags from Qualys and write to a YAML file
    $ synctags qualys get 
Create the SOT from YAML  tag list
    $ synctags qualys create
Sync the Tags from the YML Master with Qualys
    # synctags qualys sync
```    
For Crowdstrike & NinjaOne the vendor name would change but is otherwise the same set of actions (get, create, sync). Wanted to add ThreatAware as well, but doesn't appear they have added an API tro their tags :(

__THIS IS VEYR MUCH A WIP__. Use at your own risk (construcitve criticism is welcome however)


