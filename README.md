# synctags
The synctags project - syncing tags across multiple tools

Use Case: with today's proliferation of security tools, making sure teams across the org are talking about the same resources.  MOst modern security tools use Tagging.  However, every vendor implements tagging differently.

synctags uses a CLI format similar to other tool:

Get all the tags from Qualys and write to a YAML file
    $ synctags qualys get 
Create the SOT from YAML  tag list
    $ synctags qualys create
Sync the Tags from the YML Master with Qualys
    # synctags qualys sync
    
For Crowdstrike the vendor name would change

