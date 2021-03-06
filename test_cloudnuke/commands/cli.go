package commands

import (
	"fmt"
	"strings"
	"time"
	"github.com/gruntwork-io/gruntwork-cli/collections"

	"github.com/fatih/color"
	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/gruntwork-cli/shell"
	"github.com/urfave/cli"

)

// CreateCli - Create the CLI app with all commands, flags, and usage text configured.
func CreateCli(version string) *cli.App {
	app := cli.NewApp()

	app.Name = "cloud-nuke"
	app.HelpName = app.Name
	app.Author = "Gruntwork <www.gruntwork.io>"
	app.Version = version
	app.Usage = "A CLI tool to cleanup cloud resources (AWS, Azure, GCP). THIS TOOL WILL COMPLETELY REMOVE ALL RESOURCES AND ITS EFFECTS ARE IRREVERSIBLE!!!"
	app.Commands = []cli.Command{
		{
			Name:   "aws",
			Usage:  "Clean up AWS resources (ASG, ELB, ELBv2, EBS, EC2, AMI, Snapshots, ELastic IP)",
			Action: errors.WithPanicHandling(awsNuke),
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "regions to exclude",
				},
				cli.StringFlag{
					Name:  "older-than",
					Usage: "Only delete resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Skip nuke confirmation prompt. WARNING: this will automatically delete all resources without any confirmation",
				},
			},
		},
	}

	return app
}

//(jiyeonkkk) set up time to delete
func parseDurationParam(paramValue string) (*time.Time, error) {
	duration, err := time.ParseDuration(paramValue)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// make it negative so it goes back in time
	duration = -1 * duration

	//(jiyeonkkk) time - "older-than time"
	excludeAfter := time.Now().Add(duration)
	return &excludeAfter, nil
}

//(jiyeonkkk) function nuke 
func awsNuke(c *cli.Context) error {
	///(jiyeonkkk) option region
	regions := aws.GetAllRegions()
	excludedRegions := c.StringSlice("exclude-region")

	//(jiyeonkkk) check region is valid (_ means blank (i at java))
	for _, excludedRegion := range excludedRegions {
		if !collections.ListContainsElement(regions, excludedRegion) {
			return InvalidFlagError{
				Name:  "exclude-regions",
				Value: excludedRegion,
			}
		}
	}
	///(jiyeonkkk) option older-than
	excludeAfter, err := parseDurationParam(c.String("older-than"))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Logger.Infoln("Retrieving all active AWS resources")
	account, err := aws.GetAllResources(regions, excludedRegions, *excludeAfter)

	//(jiyeonkkk) ERROR		
	if err != nil {
		return errors.WithStackTrace(err)
	}

	//(jiyeonkkk) nothing to delete
	if len(account.Resources) == 0 {
		logging.Logger.Infoln("Nothing to nuke, you're all good!")
		return nil
	}

	//(jiyeonkkk) to delete
	logging.Logger.Infoln("The following AWS resources are going to be nuked: ")

	for region, resourcesInRegion := range account.Resources {
		for _, resources := range resourcesInRegion.Resources {
			for _, identifier := range resources.ResourceIdentifiers() {
				logging.Logger.Infof("* %s-%s-%s\n", resources.ResourceName(), identifier, region)
			}
		}
	}

	///(jiyeonkkk) option: not force  
	if !c.Bool("force") {
		color := color.New(color.FgHiRed, color.Bold)
		color.Println("\nTHE NEXT STEPS ARE DESTRUCTIVE AND COMPLETELY IRREVERSIBLE, PROCEED WITH CAUTION!!!")

		prompt := "\nAre you sure you want to nuke all listed resources? Enter 'nuke' to confirm: "
		//(jiyeonkkk) input on shell
		shellOptions := shell.ShellOptions{Logger: logging.Logger}
		input, err := shell.PromptUserForInput(prompt, &shellOptions)

		if err != nil {
			return errors.WithStackTrace(err)
		}

		if strings.ToLower(input) == "nuke" {
			if err := aws.NukeAllResources(account, regions); err != nil {
				return err
			}
		}
	///(jiyeonkkk) option: force 
	} else {
		logging.Logger.Infoln("The --force flag is set, so waiting for 10 seconds before proceeding to nuke everything in your account. If you don't want to proceed, hit CTRL+C now!!")
		for i := 10; i > 0; i-- {
			fmt.Printf("%d...", i)
			time.Sleep(1 * time.Second)
		}

		fmt.Println()
		if err := aws.NukeAllResources(account, regions); err != nil {
			return err
		}
	}

	///(jiyeonkkk) option: ? available add

	return nil
}

