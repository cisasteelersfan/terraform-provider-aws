package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccAWSCodeCommitRepository_importBasic(t *testing.T) {
	resName := "aws_codecommit_repository.test"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_basic(rInt),
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSCodeCommitRepository_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
				),
			},
		},
	})
}

func TestAccAWSCodeCommitRepository_withChanges(t *testing.T) {
	rInt := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "description", "This is a test description"),
				),
			},
			{
				Config: testAccCodeCommitRepository_withChanges(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "description", "This is a test description - with changes"),
				),
			},
		},
	})
}

func TestAccAWSCodeCommitRepository_create_default_branch(t *testing.T) {
	rInt := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_with_default_branch(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "default_branch", "master"),
				),
			},
		},
	})
}

func TestAccAWSCodeCommitRepository_create_and_update_default_branch(t *testing.T) {
	rInt := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckNoResourceAttr(
						"aws_codecommit_repository.test", "default_branch"),
				),
			},
			{
				Config: testAccCodeCommitRepository_with_default_branch(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "default_branch", "master"),
				),
			},
		},
	})
}

func TestAccAWSCodeCommitRepository_tags(t *testing.T) {

	rName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_codecommit_repository.test_repository",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeCommitRepositoryConfigTags1(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test_repository"),
					resource.TestCheckResourceAttr("aws_codecommit_repository.test_repository", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_codecommit_repository.test_repository", "tags.key1", "value1"),
				),
			},
			{
				Config: testAccAWSCodeCommitRepositoryConfigTags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test_repository"),
					resource.TestCheckResourceAttr("aws_codecommit_repository.test_repository", "tags.%", "2"),
					resource.TestCheckResourceAttr("aws_codecommit_repository.test_repository", "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr("aws_codecommit_repository.test_repository", "tags.key2", "value2"),
				),
			},
			{
				Config: testAccAWSCodeCommitRepositoryConfigTags1(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test_repository"),
					resource.TestCheckResourceAttr("aws_codecommit_repository.test_repository", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_codecommit_repository.test_repository", "tags.key2", "value2"),
				),
			},
		},
	})
}

func testAccCheckCodeCommitRepositoryExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		codecommitconn := testAccProvider.Meta().(*AWSClient).codecommitconn
		out, err := codecommitconn.GetRepository(&codecommit.GetRepositoryInput{
			RepositoryName: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if out.RepositoryMetadata.Arn == nil {
			return fmt.Errorf("No CodeCommit Repository Vault Found")
		}

		if *out.RepositoryMetadata.RepositoryName != rs.Primary.ID {
			return fmt.Errorf("CodeCommit Repository Mismatch - existing: %q, state: %q",
				*out.RepositoryMetadata.RepositoryName, rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckCodeCommitRepositoryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codecommitconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codecommit_repository" {
			continue
		}

		_, err := conn.GetRepository(&codecommit.GetRepositoryInput{
			RepositoryName: aws.String(rs.Primary.ID),
		})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "RepositoryDoesNotExistException" {
			continue
		}
		if err == nil {
			return fmt.Errorf("Repository still exists: %s", rs.Primary.ID)
		}
		return err
	}

	return nil
}

func testAccCodeCommitRepository_basic(rInt int) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test" {
  repository_name = "test_repository_%d"
  description     = "This is a test description"
}
`, rInt)
}

func testAccCodeCommitRepository_withChanges(rInt int) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test" {
  repository_name = "test_repository_%d"
  description     = "This is a test description - with changes"
}
`, rInt)
}

func testAccCodeCommitRepository_with_default_branch(rInt int) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test" {
  repository_name = "test_repository_%d"
  description     = "This is a test description"
  default_branch  = "master"
}
`, rInt)
}

func testAccAWSCodeCommitRepositoryConfigTags1(r, tag1Key, tag1Value string) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test_repository" {
	repository_name = "terraform-test-%s"
	tags = {
		%q = %q
	}
	}`, r, tag1Key, tag1Value)
}

func testAccAWSCodeCommitRepositoryConfigTags2(r, tag1Key, tag1Value, tag2Key, tag2Value string) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test_repository" {
	repository_name = "terraform-test-%s"
	tags = {
		%q = %q
		%q = %q
	  }
	}`, r, tag1Key, tag1Value, tag2Key, tag2Value)
}
