'use client'
import { useState } from "react";
import Link from "next/link";
import { Button } from "./ui/button";
import KAgentLogoWithText from "./kagent-logo-text";
import KagentLogo from "./kagent-logo";
import { Plus, Menu, X, ChevronDown, Brain, Server, Eye, Hammer, HomeIcon, LogOut, Ban } from "lucide-react";
import { DISABLE_MODEL_CREATION, DISABLE_MCP_SERVER_CREATION } from "@/lib/appConfig";
import { Identicon } from "./Identicon";
import { ThemeToggle } from "./ThemeToggle";
import { useUserStore } from "@/lib/userStore";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function Header() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const userId = useUserStore((state) => state.userId);
  const clearLoginSession = useUserStore((state) => state.clearLoginSession);

  const toggleMenu = () => {
    setIsMenuOpen(!isMenuOpen);
  };

  // Close mobile menu when a link inside dropdown is clicked
  const handleMobileLinkClick = () => {
    if (isMenuOpen) {
      setIsMenuOpen(false);
    }
  };

  return (
    <nav className="py-4 md:py-8 border-b">
      <div className="max-w-6xl mx-auto px-4 md:px-6">
        <div className="flex justify-between items-center">
          <Link href="/" className="flex items-center gap-3">
            <KAgentLogoWithText className="h-5" />
            <span className="text-sm font-semibold text-muted-foreground hidden lg:block">DBCI kagent Playground</span>
          </Link>
          
          {/* Mobile menu button */}
          <button 
            className="md:hidden p-2 focus:outline-none"
            onClick={toggleMenu}
            aria-label="Toggle menu"
          >
            {isMenuOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
          </button>
          
          {/* Desktop navigation */}
          <div className="hidden md:flex items-center space-x-2 lg:space-x-4">
            <Button variant="link" className="text-secondary-foreground" asChild>
              <Link href="/" className="gap-1">
                <HomeIcon className="h-4 w-4" />
                Home
              </Link>
            </Button>


            {/* Create Dropdown */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="link" className="text-secondary-foreground gap-1 px-2">
                  <Plus className="h-4 w-4" />
                  Create
                  <ChevronDown className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem asChild>
                  <Link href="/agents/new" className="gap-2 cursor-pointer w-full">
                    <KagentLogo className="h-4 w-4 text-primary" />
                    New Agent
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild={!DISABLE_MODEL_CREATION} disabled={DISABLE_MODEL_CREATION}>
                  {DISABLE_MODEL_CREATION ? (
                    <span className="gap-2 w-full flex items-center">
                      <Ban className="h-4 w-4" />
                      New Model
                    </span>
                  ) : (
                    <Link href="/models/new" className="gap-2 cursor-pointer w-full">
                      <Brain className="h-4 w-4" />
                      New Model
                    </Link>
                  )}
                </DropdownMenuItem>
                <DropdownMenuItem asChild={!DISABLE_MCP_SERVER_CREATION} disabled={DISABLE_MCP_SERVER_CREATION}>
                  {DISABLE_MCP_SERVER_CREATION ? (
                    <span className="gap-2 w-full flex items-center">
                      <Ban className="h-4 w-4" />
                      New MCP Server
                    </span>
                  ) : (
                    <Link href="/servers/new" className="gap-2 cursor-pointer w-full">
                      <Server className="h-4 w-4" />
                      New MCP Server
                    </Link>
                  )}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {/* View Dropdown */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="link" className="text-secondary-foreground gap-1 px-2">
                  View
                  <ChevronDown className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem asChild>
                  <Link href="/agents" className="gap-2 cursor-pointer w-full">
                    <KagentLogo className="h-4 w-4 text-primary" />
                    My Agents
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link href="/models" className="gap-2 cursor-pointer w-full">
                    <Brain className="h-4 w-4" />
                    Models
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link href="/tools" className="gap-2 cursor-pointer w-full">
                    <Hammer className="h-4 w-4" />
                    Tools
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link href="/servers" className="gap-2 cursor-pointer w-full">
                    <Server className="h-4 w-4" />
                    MCP Servers
                  </Link>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>



            <div className="flex items-center gap-2">
              <ThemeToggle />
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="icon" aria-label="User menu" className="overflow-hidden p-0">
                    <Identicon value={userId || "user"} size={32} />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="max-w-64">
                  <DropdownMenuItem disabled className="break-all">
                    {userId}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={clearLoginSession}>
                    <LogOut className="h-4 w-4" />
                    Sign out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </div>
        
        {/* Mobile menu */}
        {isMenuOpen && (
          <div className="md:hidden pt-4 pb-2 animate-in fade-in slide-in-from-top duration-300">
            <div className="flex flex-col space-y-1">
              {/* Mobile Home Link */}
              <Button variant="ghost" className="text-secondary-foreground justify-start px-1 gap-2" asChild>
                <Link href="/" onClick={handleMobileLinkClick}>
                  <HomeIcon className="h-4 w-4" />
                  Home
                </Link>
              </Button>

              {/* Mobile View Dropdown */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="text-secondary-foreground justify-start px-1 gap-1 w-full">
                    <Eye className="h-4 w-4" />
                    View
                    <ChevronDown className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-56">
                  <DropdownMenuItem asChild onClick={handleMobileLinkClick}>
                    <Link href="/agents" className="gap-2 cursor-pointer w-full">
                      <KagentLogo className="h-4 w-4 text-primary" />
                      My Agents
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild onClick={handleMobileLinkClick}>
                    <Link href="/models" className="gap-2 cursor-pointer w-full">
                      <Brain className="h-4 w-4" />
                      Models
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild onClick={handleMobileLinkClick}>
                    <Link href="/tools" className="gap-2 cursor-pointer w-full">
                      <Hammer className="h-4 w-4" />
                      MCP Tools
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild onClick={handleMobileLinkClick}>
                    <Link href="/servers" className="gap-2 cursor-pointer w-full">
                      <Server className="h-4 w-4" />
                      MCP Servers
                    </Link>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>

              {/* Mobile Create Dropdown */}
               <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="text-secondary-foreground justify-start px-1 gap-1 w-full">
                     <Plus className="h-4 w-4" />
                    Create
                    <ChevronDown className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-56">
                   <DropdownMenuItem asChild onClick={handleMobileLinkClick}>
                    <Link href="/agents/new" className="gap-2 cursor-pointer w-full">
                      <KagentLogo className="h-4 w-4 text-primary" />
                      New Agent
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild={!DISABLE_MODEL_CREATION} disabled={DISABLE_MODEL_CREATION} onClick={DISABLE_MODEL_CREATION ? undefined : handleMobileLinkClick}>
                    {DISABLE_MODEL_CREATION ? (
                      <span className="gap-2 w-full flex items-center">
                        <Ban className="h-4 w-4" />
                        New Model
                      </span>
                    ) : (
                      <Link href="/models/new" className="gap-2 cursor-pointer w-full">
                        <Brain className="h-4 w-4" />
                        New Model
                      </Link>
                    )}
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild={!DISABLE_MCP_SERVER_CREATION} disabled={DISABLE_MCP_SERVER_CREATION} onClick={DISABLE_MCP_SERVER_CREATION ? undefined : handleMobileLinkClick}>
                    {DISABLE_MCP_SERVER_CREATION ? (
                      <span className="gap-2 w-full flex items-center">
                        <Ban className="h-4 w-4" />
                        New MCP Server
                      </span>
                    ) : (
                      <Link href="/servers/new" className="gap-2 cursor-pointer w-full">
                        <Server className="h-4 w-4" />
                        New MCP Server
                      </Link>
                    )}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>

              <div className="flex items-center justify-end gap-2 py-2">
                <ThemeToggle />
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="icon" aria-label="User menu" className="overflow-hidden p-0">
                      <Identicon value={userId || "user"} size={32} />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="max-w-64">
                    <DropdownMenuLabel>Signed in as</DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem disabled className="break-all">
                      {userId}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => {
                        clearLoginSession()
                        handleMobileLinkClick()
                      }}
                    >
                      <LogOut className="h-4 w-4" />
                      Sign out
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          </div>
        )}
      </div>
    </nav>
  );
}
