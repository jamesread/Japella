UPDATE `cvars`
SET
  `key_name` = 'theme_color_scheme_switcher_enabled',
  `title` = 'Color scheme switcher',
  `description` = 'Show the auto/light/dark color scheme control in the PicoCrank header.',
  `category` = 'Theme'
WHERE `key_name` = 'enable_theme_color_scheme_switcher';
